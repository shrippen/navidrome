package sendspin

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
)

var _ = Describe("Track", func() {
	var (
		restore      func()
		originalOpen func(string, int) (pcmDecoder, error)
		playbackDone chan bool
		deviceKey    string
	)

	BeforeEach(func() {
		restore = configtest.SetupConfig()
		ResetRuntimesForTesting()
		conf.Server.Jukebox.Sendspin.Port = 0
		conf.Server.Jukebox.Sendspin.EnableMDNS = false
		originalOpen = openPCMDecoder
		openPCMDecoder = func(string, int) (pcmDecoder, error) {
			data := make([]int32, defaultSampleRate*defaultChannels*2) // ~2s
			for i := range data {
				data[i] = 100
			}
			return &fakeDecoder{data: data}, nil
		}
		playbackDone = make(chan bool, 1)
		deviceKey = "sendspin-track-test"
	})

	AfterEach(func() {
		openPCMDecoder = originalOpen
		ResetRuntimesForTesting()
		restore()
	})

	It("loads, plays, seeks, and pauses through the Track API", func() {
		mf := model.MediaFile{Title: "T", Path: "/virtual/track.mp3"}
		track, err := NewTrack(context.Background(), playbackDone, deviceKey, mf)
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(track.Close)

		Expect(track.IsPlaying()).To(BeFalse())
		track.SetVolume(0.8)
		track.Unpause()
		Expect(track.IsPlaying()).To(BeTrue())

		Expect(track.SetPosition(5)).To(Succeed())
		Expect(track.Position()).To(Equal(5))
		Expect(track.IsPlaying()).To(BeTrue()) // was playing; seek keeps playback

		track.Pause()
		Expect(track.IsPlaying()).To(BeFalse())
		Expect(track.SetPosition(2)).To(Succeed())
		Expect(track.Position()).To(Equal(2))
		Expect(track.IsPlaying()).To(BeFalse())
	})

	It("decodes a real fixture through ffmpeg when available", func() {
		openPCMDecoder = originalOpen
		fixture := fixturePath("test.mp3")

		dec, err := newFFmpegPCMSource(fixture, 0)
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() { _ = dec.Close() })

		// test.mp3 starts with a short silent lead-in; read enough PCM to pass it.
		buf := make([]int32, defaultSampleRate*defaultChannels/5) // 200ms
		n, err := dec.Read(buf)
		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(Equal(len(buf)))
		var nonZero bool
		for _, s := range buf {
			if s != 0 {
				nonZero = true
				break
			}
		}
		Expect(nonZero).To(BeTrue())
	})
})
