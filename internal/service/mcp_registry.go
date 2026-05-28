package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/magomedcoder/coder-server/internal/config"
	"github.com/magomedcoder/coder-server/internal/domain"
	"github.com/magomedcoder/gen/pkg/mcpclient"
	mcpdomain "github.com/magomedcoder/gen/pkg/mcpclient/domain"
)

type MCPRegistry struct {
	mu      sync.RWMutex
	servers map[int64]*mcpdomain.MCPServer
	tools   []domain.MCPToolInfo
}

func NewMCPRegistry(cfg config.MCPConfig) *MCPRegistry {
	reg := &MCPRegistry{
		servers: make(map[int64]*mcpdomain.MCPServer),
	}

	for _, s := range cfg.Servers {
		srv := toMCPServerDomain(s)
		if srv == nil {
			continue
		}
		reg.servers[srv.ID] = srv
	}

	return reg
}

func toMCPServerDomain(cfg config.MCPServerConfig) *mcpdomain.MCPServer {
	if cfg.ID <= 0 {
		return nil
	}

	enabled := true
	if cfg.Enabled != nil {
		enabled = *cfg.Enabled
	}
	if !enabled {
		return nil
	}

	name := strings.TrimSpace(cfg.Name)
	if name == "" {
		name = fmt.Sprintf("mcp-%d", cfg.ID)
	}

	headersJSON := ""
	if len(cfg.Headers) > 0 {
		b, _ := json.Marshal(cfg.Headers)
		headersJSON = string(b)
	}

	timeout := cfg.TimeoutSeconds
	if timeout <= 0 {
		timeout = 120
	}

	return &mcpdomain.MCPServer{
		ID:             cfg.ID,
		Name:           name,
		Enabled:        true,
		Transport:      strings.TrimSpace(cfg.Transport),
		URL:            strings.TrimSpace(cfg.URL),
		HeadersJSON:    headersJSON,
		TimeoutSeconds: timeout,
	}
}

func (r *MCPRegistry) Enabled() bool {
	if r == nil {
		return false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.servers) > 0
}

func (r *MCPRegistry) ServerCount() int {
	if r == nil {
		return 0
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.servers)
}

func (r *MCPRegistry) Refresh(ctx context.Context) error {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	servers := make([]*mcpdomain.MCPServer, 0, len(r.servers))
	for _, s := range r.servers {
		servers = append(servers, s)
	}
	r.mu.RUnlock()

	var tools []domain.MCPToolInfo
	for _, srv := range servers {
		declared, err := mcpclient.ListTools(ctx, srv)
		if err != nil {
			continue
		}

		for _, t := range declared {
			tools = append(tools, domain.MCPToolInfo{
				Alias:       mcpclient.ToolAlias(srv.ID, t.Name),
				ServerID:    srv.ID,
				ServerName:  srv.Name,
				Name:        t.Name,
				Description: strings.TrimSpace(t.Description),
			})
		}
	}

	r.mu.Lock()
	r.tools = tools
	r.mu.Unlock()

	return nil
}

func (r *MCPRegistry) ListTools() []domain.MCPToolInfo {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.MCPToolInfo, len(r.tools))
	copy(out, r.tools)
	return out
}

func (r *MCPRegistry) ToolsPromptBlock() string {
	tools := r.ListTools()
	if len(tools) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("MCP tools (use exact alias in calls[].tool):\n")
	for _, t := range tools {
		line := fmt.Sprintf("- %s (%s on server %q)", t.Alias, t.Name, t.ServerName)
		if t.Description != "" {
			line += ": " + t.Description
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}

	return strings.TrimSpace(b.String())
}

func (r *MCPRegistry) HasTool(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}

	if _, _, ok := mcpclient.ParseToolAlias(name); ok {
		return r.serverExistsByAlias(name)
	}

	for _, t := range r.ListTools() {
		if t.Alias == name || t.Name == name {
			return true
		}
	}

	return false
}

func (r *MCPRegistry) serverExistsByAlias(alias string) bool {
	sid, _, ok := mcpclient.ParseToolAlias(alias)
	if !ok {
		return false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok = r.servers[sid]
	return ok
}

func (r *MCPRegistry) Call(ctx context.Context, req domain.MCPCallRequest) (string, error) {
	if r == nil {
		return "", fmt.Errorf("MCP registry не инициализирован")
	}

	serverID := req.ServerID
	toolName := strings.TrimSpace(req.Name)
	if alias := strings.TrimSpace(req.Tool); alias != "" {
		sid, name, ok := mcpclient.ParseToolAlias(alias)
		if !ok {
			return "", fmt.Errorf("неизвестный MCP alias %q", alias)
		}

		serverID = sid
		toolName = name
	}

	if serverID <= 0 || toolName == "" {
		return "", fmt.Errorf("нужны tool alias или server_id+name")
	}

	r.mu.RLock()
	srv, ok := r.servers[serverID]
	r.mu.RUnlock()
	if !ok || srv == nil {
		return "", fmt.Errorf("MCP server %d не настроен", serverID)
	}

	args := req.Arguments
	if args == nil {
		args = map[string]any{}
	}

	raw, err := json.Marshal(args)
	if err != nil {
		return "", fmt.Errorf("arguments: %w", err)
	}

	return mcpclient.CallTool(ctx, srv, toolName, raw)
}
