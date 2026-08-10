package sendspin

import (
	"io"
	"sync"
	"sync/atomic"

	"github.com/Sendspin/sendspin-go/pkg/sendspin"
	"github.com/navidrome/navidrome/model"
)

const (
	defaultSampleRate = 48000
	defaultChannels   = 2
)

// switchableSource is a long-lived AudioSource that can pause, seek, change
// tracks, and apply gain while the Sendspin server keeps streaming.
type switchableSource struct {
	mu       sync.Mutex
	paused   bool
	gain     float32
	mf       *model.MediaFile
	decoder  *ffmpegPCMSource
	posFrames int64
	done     chan<- bool
	closed   atomic.Bool
}

func newSwitchableSource() *switchableSource {
	return &switchableSource{
		paused: true,
		gain:   1.0,
	}
}

func (s *switchableSource) SampleRate() int { return defaultSampleRate }
func (s *switchableSource) Channels() int   { return defaultChannels }

func (s *switchableSource) Metadata() (title, artist, album string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.mf == nil {
		return "", "", ""
	}
	return s.mf.Title, s.mf.Artist, s.mf.Album
}

func (s *switchableSource) Close() error {
	s.closed.Store(true)
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closeDecoderLocked()
}

func (s *switchableSource) Read(samples []int32) (int, error) {
	if s.closed.Load() {
		return 0, io.EOF
	}

	s.mu.Lock()
	paused := s.paused
	decoder := s.decoder
	gain := s.gain
	done := s.done
	s.mu.Unlock()

	if paused || decoder == nil {
		clear(samples)
		return len(samples), nil
	}

	n, err := decoder.Read(samples)
	if n > 0 {
		if gain != 1.0 {
			applyGain(samples[:n], gain)
		}
		frames := int64(n / defaultChannels)
		atomic.AddInt64(&s.posFrames, frames)
	}

	if err == io.EOF {
		s.mu.Lock()
		s.paused = true
		_ = s.closeDecoderLocked()
		s.mu.Unlock()
		if done != nil {
			select {
			case done <- true:
			default:
			}
		}
		clear(samples[n:])
		// Keep the Sendspin clock fed with silence until the next track loads.
		return len(samples), nil
	}
	if err != nil {
		return n, err
	}
	return n, nil
}

func (s *switchableSource) load(mf model.MediaFile, offsetSec int, done chan<- bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.closeDecoderLocked(); err != nil {
		return err
	}

	dec, err := newFFmpegPCMSource(mf.AbsolutePath(), offsetSec)
	if err != nil {
		return err
	}

	s.mf = &mf
	s.decoder = dec
	s.done = done
	s.paused = true
	atomic.StoreInt64(&s.posFrames, int64(offsetSec)*int64(defaultSampleRate))
	return nil
}

func (s *switchableSource) pause() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.paused = true
}

func (s *switchableSource) unpause() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.paused = false
}

func (s *switchableSource) setGain(gain float32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gain = gain
}

func (s *switchableSource) position() int {
	return int(atomic.LoadInt64(&s.posFrames) / int64(defaultSampleRate))
}

func (s *switchableSource) isPlaying() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.paused && s.decoder != nil
}

func (s *switchableSource) closeDecoderLocked() error {
	if s.decoder == nil {
		return nil
	}
	err := s.decoder.Close()
	s.decoder = nil
	return err
}

func applyGain(samples []int32, gain float32) {
	for i, sample := range samples {
		v := float32(sample) * gain
		if v > 8388607 {
			v = 8388607
		} else if v < -8388608 {
			v = -8388608
		}
		samples[i] = int32(v)
	}
}

// Ensure switchableSource satisfies sendspin.AudioSource at compile time.
var _ sendspin.AudioSource = (*switchableSource)(nil)
