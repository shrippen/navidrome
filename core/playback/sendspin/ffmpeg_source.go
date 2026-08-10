package sendspin

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os/exec"
	"strconv"

	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/log"
)

// ffmpegPCMSource decodes any library file to 48kHz stereo s16le PCM via ffmpeg.
type ffmpegPCMSource struct {
	cmd    *exec.Cmd
	stdout io.ReadCloser
	reader *bufio.Reader
	cancel context.CancelFunc
}

func newFFmpegPCMSource(path string, offsetSec int) (*ffmpegPCMSource, error) {
	ff := ffmpeg.New()
	cmdPath, err := ff.CmdPath()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg unavailable for Sendspin jukebox: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	args := []string{
		"-hide_banner",
		"-loglevel", "error",
	}
	if offsetSec > 0 {
		args = append(args, "-ss", strconv.Itoa(offsetSec))
	}
	args = append(args,
		"-i", path,
		"-f", "s16le",
		"-acodec", "pcm_s16le",
		"-ar", strconv.Itoa(defaultSampleRate),
		"-ac", strconv.Itoa(defaultChannels),
		"-",
	)

	cmd := exec.CommandContext(ctx, cmdPath, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("ffmpeg stdout pipe: %w", err)
	}
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("start ffmpeg: %w", err)
	}

	log.Debug("Started ffmpeg PCM decode for Sendspin", "path", path, "offset", offsetSec)

	return &ffmpegPCMSource{
		cmd:    cmd,
		stdout: stdout,
		reader: bufio.NewReaderSize(stdout, 64*1024),
		cancel: cancel,
	}, nil
}

func (s *ffmpegPCMSource) Read(samples []int32) (int, error) {
	buf := make([]byte, len(samples)*2)
	n, err := io.ReadFull(s.reader, buf)
	if n == 0 {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return 0, io.EOF
		}
		return 0, err
	}

	numSamples := n / 2
	for i := 0; i < numSamples; i++ {
		sample16 := int16(binary.LittleEndian.Uint16(buf[i*2 : i*2+2]))
		samples[i] = int32(sample16) << 8
	}

	if err == io.ErrUnexpectedEOF {
		return numSamples, io.EOF
	}
	if err != nil && err != io.EOF {
		return numSamples, err
	}
	if err == io.EOF {
		return numSamples, io.EOF
	}
	return numSamples, nil
}

func (s *ffmpegPCMSource) Close() error {
	if s.cancel != nil {
		s.cancel()
	}
	if s.stdout != nil {
		_ = s.stdout.Close()
	}
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Wait()
	}
	return nil
}
