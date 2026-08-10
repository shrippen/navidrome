package sendspin

import (
	"fmt"
	"net"
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
	mu        sync.Mutex
	source    *switchableSource
	server    *ss.Server
	onCommand CommandHandler
	started   bool
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

// ResetRuntimesForTesting stops and clears all shared runtimes. Test-only.
func ResetRuntimesForTesting() {
	runtimesMu.Lock()
	defer runtimesMu.Unlock()
	for _, rt := range runtimes {
		rt.Stop()
	}
	runtimes = map[string]*Runtime{}
}

// Start launches the Sendspin WebSocket server if it is not already running.
// If already started, the command handler is updated in place.
func (rt *Runtime) Start(onCommand CommandHandler) error {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	rt.onCommand = onCommand
	if rt.started {
		return nil
	}

	cfg := conf.Server.Jukebox.Sendspin
	port := cfg.Port
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
			rt.mu.Lock()
			handler := rt.onCommand
			rt.mu.Unlock()
			if handler != nil {
				handler(command)
			}
		},
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
	// Recreate a fresh source so a later Start is usable after Stop.
	_ = rt.source.Close()
	rt.source = newSwitchableSource()
	rt.server = nil
	rt.started = false
}

// Addr returns the bound listen address, or nil if the server is not up yet.
func (rt *Runtime) Addr() net.Addr {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.server == nil {
		return nil
	}
	return rt.server.Addr()
}

func (rt *Runtime) Source() *switchableSource {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.source
}

// Started reports whether the Sendspin server is running.
func (rt *Runtime) Started() bool {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.started
}
