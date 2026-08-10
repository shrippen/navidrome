package sendspin

import (
	"context"
	"fmt"
	"sync"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

// Track implements playback.Track using the shared Sendspin Runtime.
type Track struct {
	runtime      *Runtime
	mediaFile    model.MediaFile
	playbackDone chan bool
	mu           sync.Mutex
	closed       bool
}

// NewTrack loads a media file into the Sendspin jukebox runtime.
func NewTrack(ctx context.Context, playbackDone chan bool, deviceName string, mf model.MediaFile) (*Track, error) {
	rt := GetRuntime(deviceName)
	if err := rt.Start(nil); err != nil {
		return nil, err
	}

	t := &Track{
		runtime:      rt,
		mediaFile:    mf,
		playbackDone: playbackDone,
	}

	if err := rt.Source().load(mf, 0, playbackDone); err != nil {
		return nil, fmt.Errorf("load track into Sendspin source: %w", err)
	}

	log.Debug(ctx, "Loaded Sendspin jukebox track", "title", mf.Title, "path", mf.AbsolutePath())
	return t, nil
}

func (t *Track) String() string {
	return fmt.Sprintf("SendspinTrack(%s)", t.mediaFile.Title)
}

func (t *Track) SetVolume(value float32) {
	t.runtime.Source().setGain(value)
}

func (t *Track) Unpause() {
	t.runtime.Source().unpause()
}

func (t *Track) Pause() {
	t.runtime.Source().pause()
}

func (t *Track) Position() int {
	return t.runtime.Source().position()
}

func (t *Track) SetPosition(offset int) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return fmt.Errorf("track closed")
	}
	playing := t.runtime.Source().isPlaying()
	if err := t.runtime.Source().load(t.mediaFile, offset, t.playbackDone); err != nil {
		return err
	}
	if playing {
		t.runtime.Source().unpause()
	}
	return nil
}

func (t *Track) IsPlaying() bool {
	return t.runtime.Source().isPlaying()
}

func (t *Track) Close() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return
	}
	t.closed = true
	t.runtime.Source().pause()
}
