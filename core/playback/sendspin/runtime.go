package sendspin

import (
	"fmt"
	"sync"

	ss "github.com/Sendspin/sendspin-go/pkg/sendspin"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
)

// CommandHandler receives media commands from Sendspin controllers.
type CommandHandler func(command string)

// Runtime owns the long-lived Sendspin server and shared audio source for a
// jukebox device. Tracks load media into the source without restarting the server.
type Runtime struct {
	mu      sync.Mutex
	source  *switchableSource
	server  *ss.Server
	started bool
}

var (
	runtimesMu sync.Mutex
	runtimes   = map[string]*Runtime{}
)

// GetRuntime returns the shared Sendspin runtime for a jukebox device name.
func GetRuntime(deviceKey string) *Runtime {
	runtimesMu.Lock()
	defer runtimesMu.Unlock()
	if rt, ok := runtimes[deviceKey]; ok {
		return rt
	}
	rt := &Runtime{source: newSwitchableSource()}
	runtimes[deviceKey] = rt
	return rt
}

// Start launches the Sendspin WebSocket server if it is not already running.
func (rt *Runtime) Start(onCommand CommandHandler) error {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.started {
		return nil
	}

	cfg := conf.Server.Jukebox.Sendspin
	port := cfg.Port
	if port == 0 {
		port = 8927
	}
	name := cfg.Name
	if name == "" {
		name = "Navidrome"
	}

	server, err := ss.NewServer(ss.ServerConfig{
		Port:            port,
		Name:            name,
		Source:          rt.source,
		EnableMDNS:      cfg.EnableMDNS,
		DiscoverClients: cfg.DiscoverClients,
		SupportedRoles:  []string{"player", "metadata", "controller"},
	})
	if err != nil {
		return fmt.Errorf("create Sendspin server: %w", err)
	}

	server.Group().RegisterRole(ss.NewControllerRole(ss.ControllerConfig{
		SupportedCommands: []string{"play", "pause", "next", "previous", "stop"},
		OnCommand: func(_ *ss.ServerClient, command string) {
			if onCommand != nil {
				onCommand(command)
			}
		},
		ClockMicros: nil,
	}))

	go func() {
		if err := server.Start(); err != nil {
			log.Error("Sendspin server stopped with error", err)
		}
	}()

	rt.server = server
	rt.started = true
	log.Info("Sendspin jukebox server started", "port", port, "name", name)
	return nil
}

// Stop shuts down the Sendspin server and audio source.
func (rt *Runtime) Stop() {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if !rt.started {
		return
	}
	if rt.server != nil {
		rt.server.Stop()
	}
	_ = rt.source.Close()
	rt.started = false
}

func (rt *Runtime) Source() *switchableSource {
	return rt.source
}
