package playback

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/playback/sendspin"
	"github.com/navidrome/navidrome/model"
)

type fakeTrack struct {
	playing  bool
	volume   float32
	position int
	closed   bool
}

func (t *fakeTrack) IsPlaying() bool          { return t.playing }
func (t *fakeTrack) SetVolume(v float32)      { t.volume = v }
func (t *fakeTrack) Pause()                   { t.playing = false }
func (t *fakeTrack) Unpause()                 { t.playing = true }
func (t *fakeTrack) Position() int            { return t.position }
func (t *fakeTrack) SetPosition(offset int) error {
	t.position = offset
	return nil
}
func (t *fakeTrack) Close()                   { t.closed = true }
func (t *fakeTrack) String() string           { return "fakeTrack" }

var _ = Describe("ensureSendspinDevice", func() {
	var restore func()

	BeforeEach(func() {
		restore = configtest.SetupConfig()
	})
	AfterEach(func() {
		restore()
	})

	It("is a no-op when Sendspin is disabled", func() {
		conf.Server.Jukebox.Sendspin.Enabled = false
		devices, def := ensureSendspinDevice(nil, "")
		Expect(devices).To(BeEmpty())
		Expect(def).To(BeEmpty())
	})

	It("auto-registers a sendspin device when enabled and Devices is empty", func() {
		conf.Server.Jukebox.Sendspin.Enabled = true
		devices, def := ensureSendspinDevice(nil, "")
		Expect(devices).To(Equal([]conf.AudioDeviceDefinition{{"sendspin", sendspin.DeviceName}}))
		Expect(def).To(Equal("sendspin"))
	})

	It("does not duplicate an existing sendspin device", func() {
		conf.Server.Jukebox.Sendspin.Enabled = true
		in := []conf.AudioDeviceDefinition{{"House", "sendspin"}}
		devices, def := ensureSendspinDevice(in, "House")
		Expect(devices).To(Equal(in))
		Expect(def).To(Equal("House"))
	})

	It("does not auto-add when other devices are already configured", func() {
		conf.Server.Jukebox.Sendspin.Enabled = true
		in := []conf.AudioDeviceDefinition{{"Local", "auto"}}
		devices, def := ensureSendspinDevice(in, "")
		Expect(devices).To(Equal(in))
		Expect(def).To(BeEmpty())
	})
})

var _ = Describe("handleSendspinCommand", func() {
	var (
		ctx            context.Context
		dev            *playbackDevice
		tr             *fakeTrack
		originalCreate func(context.Context, chan bool, string, model.MediaFile) (Track, error)
	)

	BeforeEach(func() {
		ctx = context.Background()
		dev = NewPlaybackDevice(ctx, nil, "House", "sendspin")
		tr = &fakeTrack{}
		dev.ActiveTrack = tr
		dev.PlaybackQueue.Add(model.MediaFiles{
			{ID: "1", Title: "One"},
			{ID: "2", Title: "Two"},
			{ID: "3", Title: "Three"},
		})
		dev.PlaybackQueue.SetIndex(1)
		dev.User = "alice"

		originalCreate = createTrack
		createTrack = func(context.Context, chan bool, string, model.MediaFile) (Track, error) {
			next := &fakeTrack{}
			dev.ActiveTrack = next
			tr = next
			return next, nil
		}
	})

	AfterEach(func() {
		createTrack = originalCreate
	})

	It("maps play/pause onto the active track", func() {
		dev.handleSendspinCommand("play")
		Expect(tr.playing).To(BeTrue())
		dev.handleSendspinCommand("pause")
		Expect(tr.playing).To(BeFalse())
	})

	It("maps next/previous onto queue indices without losing the controlling user", func() {
		dev.handleSendspinCommand("next")
		Expect(dev.PlaybackQueue.Index).To(Equal(2))
		Expect(dev.User).To(Equal("alice"))

		dev.handleSendspinCommand("previous")
		Expect(dev.PlaybackQueue.Index).To(Equal(1))
		Expect(dev.User).To(Equal("alice"))
	})

	It("ignores next at end of queue", func() {
		dev.PlaybackQueue.SetIndex(2)
		dev.handleSendspinCommand("next")
		Expect(dev.PlaybackQueue.Index).To(Equal(2))
	})

	It("restarts the current track on previous at index 0", func() {
		dev.PlaybackQueue.SetIndex(0)
		tr.position = 30
		dev.handleSendspinCommand("previous")
		Expect(dev.PlaybackQueue.Index).To(Equal(0))
		Expect(tr.position).To(Equal(0))
	})
})
