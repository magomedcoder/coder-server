package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/magomedcoder/coder-server/internal/config"
)

type CommandSandbox struct {
	root      string
	allowed   []string
	maxOutput int
	timeout   time.Duration
}

func NewCommandSandbox(cfg config.AgentSandboxConfig, allowed []string) *CommandSandbox {
	if !cfg.Enabled || strings.TrimSpace(cfg.WorkspaceRoot) == "" {
		return nil
	}

	maxOut := cfg.MaxOutputBytes
	if maxOut <= 0 {
		maxOut = 65536
	}

	timeoutSec := cfg.CommandTimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = 120
	}

	return &CommandSandbox{
		root:      filepath.Clean(cfg.WorkspaceRoot),
		allowed:   allowed,
		maxOutput: maxOut,
		timeout:   time.Duration(timeoutSec) * time.Second,
	}
}

func (s *CommandSandbox) Run(ctx context.Context, command, cwd string) (stdout, stderr string, exitCode int, err error) {
	if s == nil {
		return "", "", -1, errors.New("sandbox не включён")
	}

	command = strings.TrimSpace(command)
	if command == "" {
		return "", "", -1, errors.New("пустая команда")
	}

	if reason := validateAllowedCommand(command, s.allowed); reason != "" {
		return "", "", -1, errors.New(reason)
	}

	workDir, err := s.resolveCwd(cwd)
	if err != nil {
		return "", "", -1, err
	}

	runCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, "/bin/sh", "-c", command)
	cmd.Dir = workDir
	cmd.Env = []string{"PATH=/usr/local/bin:/usr/bin:/bin", "HOME=" + s.root}

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	runErr := cmd.Run()
	stdout = truncateOutput(outBuf.String(), s.maxOutput)
	stderr = truncateOutput(errBuf.String(), s.maxOutput)

	exitCode = 0
	if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			return stdout, stderr, -1, runErr
		}
	}

	return stdout, stderr, exitCode, nil
}

func (s *CommandSandbox) resolveCwd(cwd string) (string, error) {
	cwd = strings.TrimSpace(cwd)
	if cwd == "" {
		return s.root, nil
	}

	clean := filepath.Clean(filepath.Join(s.root, filepath.FromSlash(strings.TrimPrefix(cwd, "/"))))
	root := s.root + string(os.PathSeparator)
	if clean != s.root && !strings.HasPrefix(clean, root) {
		return "", errors.New("cwd вне sandbox root")
	}

	info, err := os.Stat(clean)
	if err != nil {
		return "", fmt.Errorf("cwd: %w", err)
	}

	if !info.IsDir() {
		return "", errors.New("cwd не директория")
	}

	return clean, nil
}

func validateAllowedCommand(command string, allowed []string) string {
	cmd := strings.ToLower(strings.TrimSpace(command))
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return "пустая команда"
	}

	bin := fields[0]
	for _, prefix := range allowed {
		prefix = strings.ToLower(strings.TrimSpace(prefix))
		if prefix == "" {
			continue
		}

		if bin == prefix || strings.HasPrefix(bin, prefix) {
			return ""
		}
	}

	return "команда не в allowlist sandbox"
}

func truncateOutput(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}

	return s[:max] + "\n...[truncated]"
}
