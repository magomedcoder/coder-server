package service

import (
	"github.com/magomedcoder/coder-server/internal/config"
	"github.com/magomedcoder/coder-server/pkg/mcpregistry"
)

type MCPRegistry = mcpregistry.Registry

func NewMCPRegistry(cfg config.MCPConfig) *MCPRegistry {
	servers := make([]mcpregistry.ServerConfig, 0, len(cfg.Servers))
	for _, s := range cfg.Servers {
		servers = append(servers, mcpregistry.ServerConfig{
			ID:             s.ID,
			Name:           s.Name,
			Enabled:        s.Enabled,
			Transport:      s.Transport,
			URL:            s.URL,
			Headers:        s.Headers,
			TimeoutSeconds: s.TimeoutSeconds,
		})
	}
	return mcpregistry.New(servers)
}
