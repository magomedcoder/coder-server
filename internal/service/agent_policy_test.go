package service

import (
	"testing"

	"github.com/magomedcoder/coder-server/internal/domain"
)

func TestAgentPolicyBlocksCommand(t *testing.T) {
	p := NewAgentPolicy(nil, []string{"sudo "}, nil)
	calls := []domain.AgentToolCall{{
		Tool: "run_command",
		ID:   "c1",
		Args: map[string]any{"command": "sudo rm -rf /tmp/x"},
	}}
	filtered, blocked := p.FilterCalls(calls)
	if len(filtered) != 0 {
		t.Fatalf("ожидалась блокировка команды, получено: %v", filtered)
	}

	if len(blocked) != 1 {
		t.Fatalf("ожидалась одна причина блокировки, получено: %v", blocked)
	}
}

func TestAgentPolicyAllowlist(t *testing.T) {
	p := NewAgentPolicy([]string{"src/"}, nil, nil)
	calls := []domain.AgentToolCall{{
		Tool: "read_file",
		ID:   "c1",
		Args: map[string]any{"path": "src/main.go"},
	}}
	filtered, blocked := p.FilterCalls(calls)
	if len(filtered) != 1 || len(blocked) != 0 {
		t.Fatalf("ожидался разрешённый путь, filtered=%v blocked=%v", filtered, blocked)
	}
}
