// Package playback implements audio playback using PlaybackDevices. It is used to implement the Jukebox mode in turn.
// It makes use of the MPV library to do the playback. Major parts are:
// - decoder which includes decoding and transcoding of various audio file formats
// - device implementing the basic functions to work with audio devices like set, play, stop, skip, ...
// - queue a simple playlist
// - optional Sendspin backend for synchronized multi-room jukebox playback
package playback

import (
	"context"
	"fmt"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/playback/sendspin"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/singleton"
)

type PlaybackServer interface {
	Run(ctx context.Context) error
	GetDeviceForUser(user string) (*playbackDevice, error)
	GetMediaFile(id string) (*model.MediaFile, error)
	GetSendspinStatus(ctx context.Context) (SendspinStatus, error)
	SendspinCommand(ctx context.Context, command string) error
	SetSendspinGain(ctx context.Context, gain float32) error
}

type playbackServer struct {
	ctx             *context.Context
	datastore       model.DataStore
	playbackDevices []playbackDevice
}

// GetInstance returns the playback-server singleton
func GetInstance(ds model.DataStore) PlaybackServer {
	return singleton.GetInstance(func() *playbackServer {
		return &playbackServer{datastore: ds}
	})
}

// Run starts the playback server which serves request until canceled using the given context
func (ps *playbackServer) Run(ctx context.Context) error {
	ps.ctx = &ctx

	devices := conf.Server.Jukebox.Devices
	defaultDevice := conf.Server.Jukebox.Default
	devices, defaultDevice = ensureSendspinDevice(devices, defaultDevice)

	pbDevices, err := ps.initDeviceStatus(ctx, devices, defaultDevice)
	if err != nil {
		return err
	}
	ps.playbackDevices = pbDevices
	log.Info(ctx, fmt.Sprintf("%d audio devices found", len(pbDevices)))

	defaultDev, _ := ps.getDefaultDevice()

	log.Info(ctx, "Using audio device: "+defaultDev.DeviceName)

	ps.startSendspinBackends()
	defer ps.stopSendspinBackends()

	<-ctx.Done()

	// Should confirm all subprocess are terminated before returning
	return nil
}

// ensureSendspinDevice registers a Sendspin jukebox device when Jukebox.Sendspin.Enabled
// is set and no Devices entry already selects the sendspin backend.
// Auto-registration only happens when Devices is empty, so existing MPV device
// lists keep working without requiring Jukebox.Default.
func ensureSendspinDevice(devices []conf.AudioDeviceDefinition, defaultDevice string) ([]conf.AudioDeviceDefinition, string) {
	if !conf.Server.Jukebox.Sendspin.Enabled {
		return devices, defaultDevice
	}
	for _, d := range devices {
		if len(d) == 2 && sendspin.IsDevice(d[1]) {
			return devices, defaultDevice
		}
	}
	if len(devices) > 0 {
		// Caller already configured devices; require an explicit sendspin entry.
		return devices, defaultDevice
	}
	devices = []conf.AudioDeviceDefinition{{"sendspin", sendspin.DeviceName}}
	if defaultDevice == "" {
		defaultDevice = "sendspin"
	}
	return devices, defaultDevice
}

func (ps *playbackServer) startSendspinBackends() {
	for i := range ps.playbackDevices {
		dev := &ps.playbackDevices[i]
		if !sendspin.IsDevice(dev.DeviceName) {
			continue
		}
		rt := sendspin.GetRuntime(dev.DeviceName)
		if err := rt.Start(dev.handleSendspinCommand); err != nil {
			log.Error("Failed to start Sendspin jukebox backend", "device", dev.Name, err)
		}
	}
}

func (ps *playbackServer) stopSendspinBackends() {
	for i := range ps.playbackDevices {
		dev := &ps.playbackDevices[i]
		if !sendspin.IsDevice(dev.DeviceName) {
			continue
		}
		sendspin.GetRuntime(dev.DeviceName).Stop()
	}
}

func (ps *playbackServer) initDeviceStatus(ctx context.Context, devices []conf.AudioDeviceDefinition, defaultDevice string) ([]playbackDevice, error) {
	pbDevices := make([]playbackDevice, max(1, len(devices)))
	defaultDeviceFound := false

	if defaultDevice == "" {
		// if there are no devices given and no default device, we create a synthetic device named "auto"
		if len(devices) == 0 {
			pbDevices[0] = *NewPlaybackDevice(ctx, ps, "auto", "auto")
		}

		// if there is but only one entry and no default given, just use that.
		if len(devices) == 1 {
			if len(devices[0]) != 2 {
				return []playbackDevice{}, fmt.Errorf("audio device definition ought to contain 2 fields, found: %d ", len(devices[0]))
			}
			pbDevices[0] = *NewPlaybackDevice(ctx, ps, devices[0][0], devices[0][1])
		}

		if len(devices) > 1 {
			return []playbackDevice{}, fmt.Errorf("number of audio device found is %d, but no default device defined. Set Jukebox.Default", len(devices))
		}

		pbDevices[0].Default = true
		return pbDevices, nil
	}

	for idx, audioDevice := range devices {
		if len(audioDevice) != 2 {
			return []playbackDevice{}, fmt.Errorf("audio device definition ought to contain 2 fields, found: %d ", len(audioDevice))
		}

		pbDevices[idx] = *NewPlaybackDevice(ctx, ps, audioDevice[0], audioDevice[1])

		if audioDevice[0] == defaultDevice {
			pbDevices[idx].Default = true
			defaultDeviceFound = true
		}
	}

	if !defaultDeviceFound {
		return []playbackDevice{}, fmt.Errorf("default device name not found: %s ", defaultDevice)
	}
	return pbDevices, nil
}

func (ps *playbackServer) getDefaultDevice() (*playbackDevice, error) {
	for idx := range ps.playbackDevices {
		if ps.playbackDevices[idx].Default {
			return &ps.playbackDevices[idx], nil
		}
	}
	return nil, fmt.Errorf("no default device found")
}

// GetMediaFile retrieves the MediaFile given by the id parameter
func (ps *playbackServer) GetMediaFile(id string) (*model.MediaFile, error) {
	return ps.datastore.MediaFile(*ps.ctx).Get(id)
}

// GetDeviceForUser returns the audio playback device for the given user. As of now this is but only the default device.
func (ps *playbackServer) GetDeviceForUser(user string) (*playbackDevice, error) {
	log.Debug("Processing GetDevice", "user", user)
	// README: here we might plug-in the user-device mapping one fine day
	device, err := ps.getDefaultDevice()
	if err != nil {
		return nil, err
	}
	device.User = user
	return device, nil
}
