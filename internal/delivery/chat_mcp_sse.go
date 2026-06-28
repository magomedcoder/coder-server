package delivery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/magomedcoder/coder-server/internal/service"
	gendomain "github.com/magomedcoder/lmpkg/domain"
)

func writeChatMCPLoopSSE(
	ctx context.Context,
	w http.ResponseWriter,
	loop *service.ChatMCPLoop,
	messages []*gendomain.Message,
	stopSequences []string,
	timeout int32,
	genParams *gendomain.GenerationParams,
	serverIDs []int64,
	streams *ActiveStreams,
	session *service.StreamSession,
	metrics *service.Metrics,
	quota *service.TokenQuota,
	onComplete func(content string),
	sessionID string,
	requestID string,
) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "сервер не поддерживает потоковую передачу",
		})
		return
	}

	if streams != nil {
		streams.Inc()
		defer streams.Dec()
	}

	logReq(requestID, "SSE MCP-поток запущен")

	eventID := 0
	emit := func(event, data string) {
		eventID++
		ev := service.SSEEvent{ID: eventID, Event: event, Data: data}
		if session != nil {
			session.Append(ev)
		}
		fmt.Fprintf(w, "id: %d\n", ev.ID)
		fmt.Fprintf(w, "event: %s\n", event)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	onEvent := func(ev service.ChatToolEvent) {
		raw, _ := json.Marshal(ev)
		emit("tool", string(raw))
	}

	onChunk := func(chunk gendomain.LLMStreamChunk) bool {
		select {
		case <-ctx.Done():
			return false
		default:
		}

		if chunk.ReasoningContent != "" {
			delta, _ := json.Marshal(map[string]string{"text": chunk.ReasoningContent})
			emit("reasoning", string(delta))
		}

		if chunk.Content != "" {
			delta, _ := json.Marshal(map[string]string{"text": chunk.Content})
			emit("delta", string(delta))
		}

		return true
	}

	type loopResult struct {
		content   string
		reasoning string
		usage     *gendomain.StreamTokenUsage
		err       error
	}

	done := make(chan loopResult, 1)
	go func() {
		content, reasoning, usage, err := loop.Run(
			ctx,
			messages,
			stopSequences,
			timeout,
			genParams,
			serverIDs,
			requestID,
			onEvent,
			onChunk,
		)
		done <- loopResult{content, reasoning, usage, err}
	}()

	var result loopResult
	select {
	case <-ctx.Done():
		logReq(requestID, "SSE MCP отменён клиентом")
		if session != nil {
			session.MarkDone()
		}
		return
	case result = <-done:
	}

	if result.err != nil {
		logReq(requestID, "SSE MCP ошибка: %v", result.err)
		errData, _ := json.Marshal(map[string]string{"message": result.err.Error()})
		emit("error", string(errData))
		if session != nil {
			session.MarkDone()
		}

		return
	}

	endData := map[string]any{"finish": "stop"}
	if sessionID != "" {
		endData["session_id"] = sessionID
	}

	if result.usage != nil {
		endData["usage"] = map[string]int32{
			"prompt_tokens":     result.usage.PromptTokens,
			"completion_tokens": result.usage.CompletionTokens,
			"total_tokens":      result.usage.TotalTokens,
		}

		if metrics != nil {
			metrics.RecordTokens(result.usage.PromptTokens, result.usage.CompletionTokens)
		}

		if quota != nil {
			quota.Record(int64(result.usage.PromptTokens) + int64(result.usage.CompletionTokens))
		}

		logReq(requestID, "SSE MCP завершён символов=%d prompt=%d completion=%d",
			len(result.content), result.usage.PromptTokens, result.usage.CompletionTokens)
	} else {
		logReq(requestID, "SSE MCP завершён символов=%d", len(result.content))
	}

	if onComplete != nil && strings.TrimSpace(result.content) != "" {
		onComplete(result.content)
	}

	raw, _ := json.Marshal(endData)
	emit("end", string(raw))
	if session != nil {
		session.MarkDone()
	}
}
