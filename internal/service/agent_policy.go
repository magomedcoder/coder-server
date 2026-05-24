package service

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/magomedcoder/coder-server/internal/domain"
)

type AgentPolicy struct {
	allowedPaths    []string
	blockedCommands []string
}

func NewAgentPolicy(allowedPaths, blockedCommands []string) *AgentPolicy {
	return &AgentPolicy{
		allowedPaths:    allowedPaths,
		blockedCommands: blockedCommands,
	}
}

func (p *AgentPolicy) FilterCalls(calls []domain.AgentToolCall) ([]domain.AgentToolCall, []string) {
	if p == nil {
		return calls, nil
	}

	out := make([]domain.AgentToolCall, 0, len(calls))
	var blocked []string

	for _, call := range calls {
		if reason := p.validateCall(call); reason != "" {
			blocked = append(blocked, fmt.Sprintf("%s (%s): %s", call.Tool, call.ID, reason))
			continue
		}
		out = append(out, call)
	}

	return out, blocked
}

func (p *AgentPolicy) validateCall(call domain.AgentToolCall) string {
	switch call.Tool {
	case "run_command":
		return p.validateCommand(call.Args)
	case "list_dir", "glob_search":
		return p.validatePathArg(call.Args, "dir")
	default:
		return p.validatePathArg(call.Args, "path")
	}
}

func (p *AgentPolicy) validateCommand(args map[string]any) string {
	cmd := strings.ToLower(strings.TrimSpace(argString(args, "command")))
	if cmd == "" {
		return "пустая команда"
	}
	for _, blocked := range p.blockedCommands {
		if strings.Contains(cmd, strings.ToLower(blocked)) {
			return "команда заблокирована политикой"
		}
	}
	return ""
}

func (p *AgentPolicy) validatePathArg(args map[string]any, key string) string {
	if len(p.allowedPaths) == 0 {
		return ""
	}

	path := filepath.ToSlash(strings.TrimSpace(argString(args, key)))
	if path == "" {
		return ""
	}

	if strings.HasPrefix(path, "../") || strings.Contains(path, "/../") || strings.HasPrefix(path, "/") {
		if !p.absoluteAllowed(path) {
			if strings.HasPrefix(path, "../") || strings.Contains(path, "/../") {
				return "path traversal запрещён"
			}
		}
	}

	for _, prefix := range p.allowedPaths {
		prefix = filepath.ToSlash(strings.TrimPrefix(strings.TrimSpace(prefix), "./"))
		prefix = strings.TrimSuffix(prefix, "/")
		if prefix == "" {
			continue
		}
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return ""
		}
	}

	return "путь вне allowlist"
}

func (p *AgentPolicy) absoluteAllowed(path string) bool {
	for _, prefix := range p.allowedPaths {
		prefix = filepath.ToSlash(strings.TrimSpace(prefix))
		if prefix != "" && strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func argString(args map[string]any, key string) string {
	if args == nil {
		return ""
	}

	v, ok := args[key]
	if !ok || v == nil {
		return ""
	}

	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}
