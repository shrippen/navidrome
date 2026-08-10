package sendspin

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/Sendspin/sendspin-go/pkg/protocol"
	"github.com/gorilla/websocket"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
)

func waitForRuntimePort(rt *Runtime) int {
	GinkgoHelper()
	deadline := time.Now().Add(3 * time.Second)
	for rt.Addr() == nil {
		Expect(time.Now().Before(deadline)).To(BeTrue(), "runtime did not bind within timeout")
		time.Sleep(10 * time.Millisecond)
	}
	return rt.Addr().(*net.TCPAddr).Port
}

var _ = Describe("Runtime integration", func() {
	var restore func()

	BeforeEach(func() {
		restore = configtest.SetupConfig()
		ResetRuntimesForTesting()
		conf.Server.Jukebox.Sendspin.Port = 0
		conf.Server.Jukebox.Sendspin.EnableMDNS = false
		conf.Server.Jukebox.Sendspin.DiscoverClients = false
		conf.Server.Jukebox.Sendspin.Name = "Navidrome Test"
	})

	AfterEach(func() {
		ResetRuntimesForTesting()
		restore()
	})

	It("starts, accepts a Sendspin client hello, and streams audio chunks", func() {
		fixture := fixturePath("test.mp3")

		rt := GetRuntime("sendspin-integration")
		var commands []string
		Expect(rt.Start(func(cmd string) { commands = append(commands, cmd) })).To(Succeed())
		Expect(rt.Started()).To(BeTrue())

		done := make(chan bool, 1)
		mf := model.MediaFile{
			Title:  "Test Track",
			Artist: "Fixture",
			Album:  "Fixtures",
			Path:   fixture,
		}
		Expect(rt.Source().load(mf, 0, done)).To(Succeed())
		rt.Source().unpause()

		port := waitForRuntimePort(rt)
		wsURL := fmt.Sprintf("ws://127.0.0.1:%d/sendspin", port)
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() { _ = conn.Close() })

		hello := protocol.Message{
			Type: "client/hello",
			Payload: protocol.ClientHello{
				ClientID:       "navidrome-test-client",
				Name:           "Navidrome Test Client",
				Version:        1,
				SupportedRoles: []string{"player@v1", "metadata@v1", "controller@v1"},
				PlayerV1Support: &protocol.PlayerV1Support{
					SupportedFormats: []protocol.AudioFormat{{
						Codec:      "pcm",
						Channels:   2,
						SampleRate: 48000,
						BitDepth:   24,
					}},
					BufferCapacity: 1_048_576,
				},
			},
		}
		Expect(conn.WriteJSON(hello)).To(Succeed())

		Expect(conn.SetReadDeadline(time.Now().Add(5 * time.Second))).To(Succeed())
		var msg protocol.Message
		Expect(conn.ReadJSON(&msg)).To(Succeed())
		Expect(msg.Type).To(Equal("server/hello"))

		// Drain control/silent lead-in chunks until we see non-silent PCM.
		var sawAudio bool
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			Expect(conn.SetReadDeadline(time.Now().Add(2 * time.Second))).To(Succeed())
			msgType, data, err := conn.ReadMessage()
			if err != nil {
				continue
			}
			if msgType == websocket.BinaryMessage {
				Expect(len(data)).To(BeNumerically(">", 9))
				Expect(data[0]).To(Equal(byte(4))) // player audio chunk type
				payload := data[9:]
				for _, b := range payload {
					if b != 0 {
						sawAudio = true
						break
					}
				}
				if sawAudio {
					break
				}
				continue
			}
			var ctrl protocol.Message
			_ = json.Unmarshal(data, &ctrl)
		}
		Expect(sawAudio).To(BeTrue(), "expected non-silent audio from Sendspin server")
		Eventually(func() int { return rt.Source().position() }, "2s").Should(BeNumerically(">", 0))

		rt.Stop()
		Expect(rt.Started()).To(BeFalse())
	})

	It("updates the command handler after Start", func() {
		rt := GetRuntime("sendspin-handler")
		Expect(rt.Start(nil)).To(Succeed())
		called := make(chan string, 1)
		Expect(rt.Start(func(cmd string) { called <- cmd })).To(Succeed())

		rt.mu.Lock()
		handler := rt.onCommand
		rt.mu.Unlock()
		Expect(handler).ToNot(BeNil())
		handler("pause")
		Eventually(called).Should(Receive(Equal("pause")))
	})
})
