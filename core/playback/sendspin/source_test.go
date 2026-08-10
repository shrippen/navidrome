package sendspin

import (
	"io"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/navidrome/navidrome/model"
)

type fakeDecoder struct {
	data   []int32
	pos    int
	closed atomic.Bool
}

func (d *fakeDecoder) Read(samples []int32) (int, error) {
	if d.closed.Load() {
		return 0, io.EOF
	}
	if d.pos >= len(d.data) {
		return 0, io.EOF
	}
	n := copy(samples, d.data[d.pos:])
	d.pos += n
	if d.pos >= len(d.data) {
		return n, io.EOF
	}
	return n, nil
}

func (d *fakeDecoder) Close() error {
	d.closed.Store(true)
	return nil
}

var _ = Describe("applyGain", func() {
	It("scales samples and clamps to 24-bit range", func() {
		samples := []int32{1000, -1000, 8388607}
		applyGain(samples, 0.5)
		Expect(samples[0]).To(Equal(int32(500)))
		Expect(samples[1]).To(Equal(int32(-500)))
		Expect(samples[2]).To(Equal(int32(4194303)))

		hot := []int32{8388607}
		applyGain(hot, 2.0)
		Expect(hot[0]).To(Equal(int32(8388607)))
	})
})

var _ = Describe("switchableSource", func() {
	var src *switchableSource
	var originalOpen func(string, int) (pcmDecoder, error)

	BeforeEach(func() {
		src = newSwitchableSource()
		originalOpen = openPCMDecoder
	})

	AfterEach(func() {
		_ = src.Close()
		openPCMDecoder = originalOpen
	})

	It("returns silence while paused", func() {
		buf := make([]int32, 8)
		n, err := src.Read(buf)
		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(Equal(8))
		Expect(buf).To(Equal(make([]int32, 8)))
	})

	It("reports empty metadata without a loaded track", func() {
		title, artist, album := src.Metadata()
		Expect(title).To(BeEmpty())
		Expect(artist).To(BeEmpty())
		Expect(album).To(BeEmpty())
	})

	It("plays decoded samples with gain and advances position", func() {
		openPCMDecoder = func(path string, offsetSec int) (pcmDecoder, error) {
			Expect(path).To(Equal("/music/track.mp3"))
			Expect(offsetSec).To(Equal(0))
			// 2 channels * 48000 samples = 1 second of frames worth of ints
			data := make([]int32, defaultSampleRate*defaultChannels)
			for i := range data {
				data[i] = 2000
			}
			return &fakeDecoder{data: data}, nil
		}

		mf := model.MediaFile{Title: "Song", Artist: "Artist", Album: "Album", Path: "/music/track.mp3"}
		Expect(src.load(mf, 0, nil)).To(Succeed())
		title, artist, album := src.Metadata()
		Expect(title).To(Equal("Song"))
		Expect(artist).To(Equal("Artist"))
		Expect(album).To(Equal("Album"))

		src.setGain(0.5)
		src.unpause()
		Expect(src.isPlaying()).To(BeTrue())

		buf := make([]int32, 100)
		n, err := src.Read(buf)
		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(Equal(100))
		Expect(buf[0]).To(Equal(int32(1000)))
		Expect(src.position()).To(Equal(0)) // less than 1s of frames consumed
	})

	It("signals playbackDone and returns silence after decoder EOF", func() {
		openPCMDecoder = func(string, int) (pcmDecoder, error) {
			return &fakeDecoder{data: []int32{10, 20}}, nil
		}
		done := make(chan bool, 1)
		mf := model.MediaFile{Title: "Short", Path: "/music/short.mp3"}
		Expect(src.load(mf, 0, done)).To(Succeed())
		src.unpause()

		buf := make([]int32, 8)
		n, err := src.Read(buf)
		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(Equal(8))
		Eventually(done).Should(Receive(BeTrue()))
		Expect(src.isPlaying()).To(BeFalse())
	})

	It("seeks by reopening the decoder at an offset", func() {
		var gotOffset int
		openPCMDecoder = func(path string, offsetSec int) (pcmDecoder, error) {
			gotOffset = offsetSec
			return &fakeDecoder{data: []int32{1, 2, 3, 4}}, nil
		}
		mf := model.MediaFile{Title: "Seek", Path: "/music/seek.mp3"}
		Expect(src.load(mf, 12, nil)).To(Succeed())
		Expect(gotOffset).To(Equal(12))
		Expect(src.position()).To(Equal(12))
	})
})
