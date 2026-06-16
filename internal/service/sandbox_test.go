package service

import (
	"testing"

	"github.com/magomedcoder/coder-server/internal/config"
)

func boolPtr(v bool) *bool {
	return &v
}

func TestCommandSandboxAllowed(t *testing.T) {
	root := t.TempDir()
	sb := NewCommandSandbox(config.AgentSandboxConfig{
		Enabled:           boolPtr(true),
		WorkspaceRoot:     root,
		MaxOutputBytes:    1024,
		CommandTimeoutSec: 5,
	}, []string{"echo"})
	if sb == nil {
		t.Fatal("ожидался sandbox")
	}

	stdout, _, code, err := sb.Run(t.Context(), "echo hello", "")
	if err != nil || code != 0 {
		t.Fatalf("запуск не удался: code=%d err=%v stdout=%q", code, err, stdout)
	}
	if stdout != "hello\n" && stdout != "hello" {
		t.Fatalf("неожиданный stdout %q", stdout)
	}
}

func TestCommandSandboxBlocksUnknown(t *testing.T) {
	root := t.TempDir()
	sb := NewCommandSandbox(config.AgentSandboxConfig{
		Enabled:       boolPtr(true),
		WorkspaceRoot: root,
	}, []string{"go"})
	_, _, _, err := sb.Run(t.Context(), "curl example.com", "")
	if err == nil {
		t.Fatal("ожидалась блокировка команды")
	}
}
