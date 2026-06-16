package mcpregistry

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/magomedcoder/gen/pkg/domain"
	"github.com/magomedcoder/gen/pkg/mcpclient"
	mcpdomain "github.com/magomedcoder/gen/pkg/mcpclient/domain"
	"github.com/magomedcoder/gen/pkg/toolloop"
)

type Registry struct {
	mu         sync.RWMutex
	servers    map[int64]*mcpdomain.MCPServer
	tools      []ToolInfo
	toolsCache *mcpclient.ToolsListCache
}

func New(servers []ServerConfig) *Registry {
	cache := mcpclient.NewToolsListCache()
	mcpclient.SetToolsListCacheForNotifications(cache)

	reg := &Registry{
		servers:    make(map[int64]*mcpdomain.MCPServer),
		toolsCache: cache,
	}

	for _, s := range servers {
		srv := toMCPServerDomain(s)
		if srv == nil {
			continue
		}

		reg.servers[srv.ID] = srv
	}

	return reg
}

func toMCPServerDomain(cfg ServerConfig) *mcpdomain.MCPServer {
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

	srv := &mcpdomain.MCPServer{
		ID:             cfg.ID,
		Name:           name,
		Enabled:        true,
		Transport:      strings.TrimSpace(cfg.Transport),
		URL:            strings.TrimSpace(cfg.URL),
		HeadersJSON:    headersJSON,
		TimeoutSeconds: timeout,
	}

	mcpdomain.NormalizeMCPServer(srv)
	if err := mcpdomain.ValidateMCPServerStructure(srv); err != nil {
		log.Printf("MCP server %d (%q) пропущен: %v", cfg.ID, name, err)
		return nil
	}

	return srv
}

func (r *Registry) Enabled() bool {
	if r == nil {
		return false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.servers) > 0
}

func (r *Registry) ServerCount() int {
	if r == nil {
		return 0
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.servers)
}

func (r *Registry) Refresh(ctx context.Context) error {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	servers := make([]*mcpdomain.MCPServer, 0, len(r.servers))
	for _, s := range r.servers {
		servers = append(servers, s)
	}

	r.mu.RUnlock()

	var tools []ToolInfo
	for _, srv := range servers {
		declared, err := r.toolsCache.ListToolsCached(ctx, srv, mcpclient.DefaultToolsListCacheTTL)
		if err != nil {
			continue
		}

		for _, t := range declared {
			tools = append(tools, ToolInfo{
				Alias:          mcpclient.ToolAlias(srv.ID, t.Name),
				ServerID:       srv.ID,
				ServerName:     srv.Name,
				Name:           t.Name,
				Description:    strings.TrimSpace(t.Description),
				ParametersJSON: t.ParametersJSON,
			})
		}
	}

	r.mu.Lock()
	r.tools = tools
	r.mu.Unlock()

	return nil
}

func (r *Registry) ListTools() []ToolInfo {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ToolInfo, len(r.tools))
	copy(out, r.tools)
	return out
}

func (r *Registry) ToolsPromptBlock() string {
	return r.ToolsPromptBlockForServers(nil)
}

func (r *Registry) ToolsPromptBlockForServers(serverIDs []int64) string {
	tools := r.ListTools()
	if len(tools) == 0 {
		return ""
	}

	allowed := make(map[int64]struct{}, len(serverIDs))
	for _, id := range serverIDs {
		if id > 0 {
			allowed[id] = struct{}{}
		}
	}
	filterAll := len(allowed) == 0

	var b strings.Builder
	b.WriteString("MCP tools (use exact alias in calls[].tool):\n")
	for _, t := range tools {
		if !filterAll {
			if _, ok := allowed[t.ServerID]; !ok {
				continue
			}
		}
		line := fmt.Sprintf("- %s (%s on server %q)", t.Alias, t.Name, t.ServerName)
		if t.Description != "" {
			line += ": " + t.Description
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}

	return strings.TrimSpace(b.String())
}

func (r *Registry) HasTool(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}

	if _, _, ok := mcpclient.ParseToolAlias(name); ok {
		return r.ServerExistsByAlias(name)
	}

	for _, t := range r.ListTools() {
		if t.Alias == name || t.Name == name {
			return true
		}
	}

	return false
}

func (r *Registry) ServerExistsByAlias(alias string) bool {
	sid, _, ok := mcpclient.ParseToolAlias(alias)
	if !ok {
		return false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok = r.servers[sid]
	return ok
}

func (r *Registry) Call(ctx context.Context, req CallRequest) (string, error) {
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

	return mcpclient.CallTool(ctx, srv, toolName, raw, r.toolsCache)
}

func (r *Registry) GenerationTools(ctx context.Context, serverIDs []int64) ([]domain.Tool, error) {
	if r == nil {
		return nil, fmt.Errorf("MCP registry не инициализирован")
	}

	_ = r.Refresh(ctx)
	tools := r.ListTools()
	if len(tools) == 0 {
		return nil, nil
	}

	allowed := make(map[int64]struct{}, len(serverIDs))
	for _, id := range serverIDs {
		if id > 0 {
			allowed[id] = struct{}{}
		}
	}
	filterAll := len(allowed) == 0

	allowedNames := make(map[string]struct{})
	var out []domain.Tool
	for _, t := range tools {
		if !filterAll {
			if _, ok := allowed[t.ServerID]; !ok {
				continue
			}
		}

		n := toolloop.NormalizeToolName(t.Alias)
		if _, dup := allowedNames[n]; dup {
			continue
		}
		allowedNames[n] = struct{}{}

		desc := strings.TrimSpace(t.Description)
		if t.ServerName != "" {
			desc = "[MCP " + t.ServerName + "] " + desc
		}

		out = append(out, domain.Tool{
			Name:           t.Alias,
			Description:    strings.TrimSpace(desc),
			ParametersJSON: t.ParametersJSON,
		})
	}

	return out, nil
}
