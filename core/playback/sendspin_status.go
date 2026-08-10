package playback

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/playback/sendspin"
)

// SendspinClient describes a connected Sendspin player/controller for the UI.
type SendspinClient struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	State  string `json:"state"`
	Volume int    `json:"volume"`
	Muted  bool   `json:"muted"`
	Codec  string `json:"codec"`
}

// SendspinNowPlaying is the track currently loaded into the Sendspin source.
type SendspinNowPlaying struct {
	Title  string `json:"title"`
	Artist string `json:"artist"`
	Album  string `json:"album"`
}

// SendspinStatus is the UI-facing snapshot of the Sendspin jukebox backend.
type SendspinStatus struct {
	Enabled       bool                `json:"enabled"`
	Running       bool                `json:"running"`
	ServerName    string              `json:"serverName"`
	Port          int                 `json:"port"`
	ConnectionURL string              `json:"connectionUrl"`
	Playing       bool                `json:"playing"`
	Gain          float32             `json:"gain"`
	QueueIndex    int                 `json:"queueIndex"`
	QueueSize     int                 `json:"queueSize"`
	NowPlaying    *SendspinNowPlaying `json:"nowPlaying,omitempty"`
	Clients       []SendspinClient    `json:"clients"`
}

// UIEnabled reports whether the Sendspin jukebox UI should be offered.
func SendspinUIEnabled() bool {
	if !conf.Server.Jukebox.Enabled {
		return false
	}
	if conf.Server.Jukebox.Sendspin.Enabled {
		return true
	}
	for _, d := range conf.Server.Jukebox.Devices {
		if len(d) == 2 && sendspin.IsDevice(d[1]) {
			return true
		}
	}
	return false
}

func (ps *playbackServer) GetSendspinStatus(_ context.Context) (SendspinStatus, error) {
	status := SendspinStatus{
		Enabled:    SendspinUIEnabled(),
		ServerName: conf.Server.Jukebox.Sendspin.Name,
		Port:       conf.Server.Jukebox.Sendspin.Port,
		Clients:    []SendspinClient{},
	}
	if status.ServerName == "" {
		status.ServerName = "Navidrome"
	}
	if status.Port == 0 {
		status.Port = 8927
	}
	if !status.Enabled {
		return status, nil
	}

	var dev *playbackDevice
	if d, err := ps.getDefaultDevice(); err == nil && sendspin.IsDevice(d.DeviceName) {
		dev = d
	} else {
		for i := range ps.playbackDevices {
			if sendspin.IsDevice(ps.playbackDevices[i].DeviceName) {
				dev = &ps.playbackDevices[i]
				break
			}
		}
	}
	if dev != nil {
		st := dev.getStatus()
		status.Playing = st.Playing
		status.Gain = st.Gain
		status.QueueIndex = st.CurrentIndex
		status.QueueSize = dev.PlaybackQueue.Size()
	}

	deviceKey := sendspin.DeviceName
	if keys := sendspin.ActiveRuntimes(); len(keys) > 0 {
		deviceKey = keys[0]
	}
	rt := sendspin.GetRuntime(deviceKey)
	status.Running = rt.Started()
	if addr := rt.Addr(); addr != nil {
		status.ConnectionURL = connectionURL(addr, status.Port)
		if ta, ok := addr.(*net.TCPAddr); ok && ta.Port != 0 {
			status.Port = ta.Port
		}
	} else {
		status.ConnectionURL = fmt.Sprintf("ws://127.0.0.1:%d/sendspin", status.Port)
	}

	title, artist, album := rt.Source().Metadata()
	if title != "" || artist != "" || album != "" {
		status.NowPlaying = &SendspinNowPlaying{Title: title, Artist: artist, Album: album}
	}

	for _, c := range rt.Clients() {
		status.Clients = append(status.Clients, SendspinClient{
			ID:     c.ID,
			Name:   c.Name,
			State:  c.State,
			Volume: c.Volume,
			Muted:  c.Muted,
			Codec:  c.Codec,
		})
	}
	return status, nil
}

func connectionURL(addr net.Addr, fallbackPort int) string {
	host := "127.0.0.1"
	port := fallbackPort
	if ta, ok := addr.(*net.TCPAddr); ok {
		if ta.Port != 0 {
			port = ta.Port
		}
		if ta.IP != nil && !ta.IP.IsUnspecified() {
			host = ta.IP.String()
		}
	}
	return fmt.Sprintf("ws://%s:%d/sendspin", host, port)
}

func (ps *playbackServer) SendspinCommand(ctx context.Context, command string) error {
	dev, err := ps.sendspinDevice()
	if err != nil {
		return err
	}
	dev.handleSendspinCommand(strings.TrimSpace(strings.ToLower(command)))
	return nil
}

func (ps *playbackServer) SetSendspinGain(ctx context.Context, gain float32) error {
	dev, err := ps.sendspinDevice()
	if err != nil {
		return err
	}
	if gain < 0 {
		gain = 0
	}
	if gain > 1 {
		gain = 1
	}
	_, err = dev.SetGain(ctx, gain)
	return err
}

func (ps *playbackServer) sendspinDevice() (*playbackDevice, error) {
	for i := range ps.playbackDevices {
		if sendspin.IsDevice(ps.playbackDevices[i].DeviceName) {
			return &ps.playbackDevices[i], nil
		}
	}
	return nil, fmt.Errorf("no Sendspin jukebox device configured")
}
