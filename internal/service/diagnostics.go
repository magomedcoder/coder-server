package service

import (
	"fmt"
	"strings"

	"github.com/magomedcoder/coder-server/internal/domain"
)

func FormatObservations(obs []domain.AgentStepObservation) string {
	if len(obs) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("Observations from previous tool calls:\n")

	for _, o := range obs {
		if !o.OK || o.Error != "" {
			b.WriteString("FAILED: ")
			if hint := SummarizeFailure(o); hint != "" {
				b.WriteString(hint)
				b.WriteByte('\n')
			}
		}
		line := observationLine(o)
		b.WriteString(line)
		b.WriteByte('\n')
	}

	return b.String()
}

func observationLine(o domain.AgentStepObservation) string {
	parts := []string{o.Tool}
	if o.CallID != "" {
		parts = append(parts, "id="+o.CallID)
	}

	if !o.OK {
		parts = append(parts, "ok=false")
	}

	if o.Error != "" {
		parts = append(parts, "error="+o.Error)
	}

	if o.Result != nil {
		if stderr, ok := o.Result["stderr"].(string); ok && stderr != "" {
			parts = append(parts, "stderr="+truncate(stderr, 500))
		}

		if exitCode, ok := o.Result["exit_code"]; ok {
			parts = append(parts, fmt.Sprintf("exit_code=%v", exitCode))
		}
	}

	return strings.Join(parts, " ")
}

func SummarizeFailure(o domain.AgentStepObservation) string {
	if o.Error != "" {
		return o.Error
	}

	if o.Result == nil {
		return "tool call failed"
	}

	if stderr, ok := o.Result["stderr"].(string); ok && stderr != "" {
		if line := extractTracebackLine(stderr); line != "" {
			return line
		}
		return truncate(strings.TrimSpace(stderr), 300)
	}

	if msg, ok := o.Result["message"].(string); ok && msg != "" {
		return truncate(msg, 300)
	}

	return "tool call failed without details"
}

func extractTracebackLine(stderr string) string {
	lines := strings.Split(stderr, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		if strings.Contains(lower, "error") ||
			strings.HasPrefix(line, "panic:") ||
			strings.Contains(line, "Exception") ||
			strings.Contains(line, "SyntaxError") ||
			strings.Contains(line, "Compil") {
			return truncate(line, 300)
		}
	}
	return ""
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}

	return s[:max] + "..."
}
