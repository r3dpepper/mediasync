package discovery

import (
	"fmt"
	"os"

	"github.com/hashicorp/mdns"
	"github.com/rs/zerolog/log"
)

type MDNSServer struct {
	server *mdns.Server
}

func StartMDNS(serviceName string, port int) (*MDNSServer, error) {
	_, err := os.Hostname()
	if err != nil {
		// hostname = "media-server"
	}

	// Service instance info
	info := []string{
		"version=1.0.0",
		"type=media-server",
	}

	// Create mDNS service
	service, err := mdns.NewMDNSService(
		serviceName,         // Instance name
		"_mediaserver._tcp", // Service type
		"",                  // Domain (empty = .local)
		"",                  // Host (empty = auto-detect)
		port,                // Port
		nil,                 // IPs (nil = auto-detect)
		info,                // TXT records
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create mDNS service: %w", err)
	}

	// Start mDNS server
	server, err := mdns.NewServer(&mdns.Config{Zone: service})
	if err != nil {
		return nil, fmt.Errorf("failed to start mDNS server: %w", err)
	}

	log.Info().
		Str("service", serviceName+".local").
		Str("type", "_mediaserver._tcp").
		Int("port", port).
		Msg("mDNS service advertised")

	return &MDNSServer{server: server}, nil
}

func (m *MDNSServer) Shutdown() error {
	if m.server != nil {
		return m.server.Shutdown()
	}
	return nil
}
