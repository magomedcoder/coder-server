package mcpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync/atomic"
	"unicode/utf8"

	"github.com/magomedcoder/gen/pkg/logger"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const maxMCPNotificationLogRunes = 2048

var logServerMessages atomic.Bool

var bearerLikeRE = regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9\-_\.~+/=]{8,}`)

func redactMCPLogMessagePayload(s string) string {
	if s == "" {
		return s
	}
	return bearerLikeRE.ReplaceAllString(s, "bearer [REDACTED]")
}

func SetLogServerMessages(v bool) {
	logServerMessages.Store(v)
}

func LogServerMessages() bool {
	return logServerMessages.Load()
}

func truncateRunesStr(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	i := 0
	for j, w := 0, 0; j < len(s); j += w {
		_, w = utf8.DecodeRuneInString(s[j:])
		if i >= maxRunes {
			return s[:j] + "..."
		}
		i++
	}
	return s
}

func formatLoggingMessageData(data any) string {
	if data == nil {
		return ""
	}
	switch x := data.(type) {
	case string:
		s := redactMCPLogMessagePayload(strings.TrimSpace(x))
		return truncateRunesStr(s, maxMCPNotificationLogRunes)
	default:
		b, err := json.Marshal(x)
		if err != nil {
			s := redactMCPLogMessagePayload(fmt.Sprintf("%v", x))
			return truncateRunesStr(s, maxMCPNotificationLogRunes)
		}
		s := redactMCPLogMessagePayload(string(b))
		return truncateRunesStr(s, maxMCPNotificationLogRunes)
	}
}

func loggingMessageHandlerForServer(srvName string, serverID int64) func(context.Context, *mcp.LoggingMessageRequest) {
	return func(_ context.Context, req *mcp.LoggingMessageRequest) {
		if req == nil || req.Params == nil {
			return
		}
		p := req.Params
		msg := formatLoggingMessageData(p.Data)
		var b strings.Builder
		fmt.Fprintf(&b, "MCP notify/message server_id=%d", serverID)
		if t := strings.TrimSpace(srvName); t != "" {
			fmt.Fprintf(&b, " name=%q", t)
		}
		if lg := strings.TrimSpace(p.Logger); lg != "" {
			fmt.Fprintf(&b, " logger=%q", lg)
		}
		fmt.Fprintf(&b, " level=%s", p.Level)
		if msg != "" {
			b.WriteString(" ")
			b.WriteString(msg)
		}
		line := b.String()
		switch strings.ToLower(string(p.Level)) {
		case "emergency", "alert", "critical", "error":
			logger.E("%s", line)
		case "warning":
			logger.W("%s", line)
		case "notice", "info":
			logger.I("%s", line)
		case "debug":
			logger.D("%s", line)
		default:
			logger.V("%s", line)
		}
	}
}
