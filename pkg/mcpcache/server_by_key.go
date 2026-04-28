package mcpcache

import (
	"strings"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ServerBuilder func(key string) *mcp.Server

type ServerByKey struct {
	mu      sync.Mutex
	servers map[string]*mcp.Server
	build   ServerBuilder
}

func NewServerByKey(builder ServerBuilder) *ServerByKey {
	return &ServerByKey{
		servers: map[string]*mcp.Server{},
		build:   builder,
	}
}

func (c *ServerByKey) Get(key string) *mcp.Server {
	normalizedKey := strings.TrimSpace(key)

	c.mu.Lock()
	defer c.mu.Unlock()

	if srv, ok := c.servers[normalizedKey]; ok {
		return srv
	}

	srv := c.build(normalizedKey)
	c.servers[normalizedKey] = srv
	return srv
}
