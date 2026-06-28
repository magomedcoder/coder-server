package delivery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/magomedcoder/coder-server/internal/service"
	"github.com/magomedcoder/coder-server/pkg/llmclient"
	gendomain "github.com/magomedcoder/lmpkg/domain"
)

func writeRunnerSSE(
	ctx context.Context,
	w http.ResponseWriter,
	streamMeta llmclient.StreamMeta,
	chunks <-chan gendomain.LLMStreamChunk,
	streams *ActiveStreams,
	session *service.StreamSession,
	metrics *service.Metrics,
	quota *service.TokenQuota,
	sessionID string,
	requestID string,
	onComplete func(content string),
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

	eventID := 0
	var usage *gendomain.StreamTokenUsage
	var full strings.Builder

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

	if streamMeta.RunnerAttempts > 0 {
		data, _ := json.Marshal(map[string]any{
			"phase":   "retrying",
			"attempt": streamMeta.RunnerAttempts + 1,
			"runner":  streamMeta.RunnerAddr,
		})
		emit("status", string(data))
	}

	for {
		select {
		case <-ctx.Done():
			logReq(requestID, "SSE отменён клиентом символов=%d", full.Len())
			if session != nil {
				session.MarkDone()
			}
			return
		case chunk, ok := <-chunks:
			if !ok {
				endData := map[string]any{"finish": "stop"}
				if sessionID != "" {
					endData["session_id"] = sessionID
				}
				if usage != nil {
					endData["usage"] = map[string]int32{
						"prompt_tokens":     usage.PromptTokens,
						"completion_tokens": usage.CompletionTokens,
						"total_tokens":      usage.TotalTokens,
					}
					if metrics != nil {
						metrics.RecordTokens(usage.PromptTokens, usage.CompletionTokens)
					}
					if quota != nil {
						quota.Record(int64(usage.PromptTokens) + int64(usage.CompletionTokens))
					}
					logReq(requestID, "SSE завершён символов=%d prompt=%d completion=%d", full.Len(), usage.PromptTokens, usage.CompletionTokens)
				} else {
					logReq(requestID, "SSE завершён символов=%d", full.Len())
				}
				raw, _ := json.Marshal(endData)
				emit("end", string(raw))
				if onComplete != nil {
					onComplete(full.String())
				}
				if session != nil {
					session.MarkDone()
				}
				return
			}

			if chunk.Usage != nil {
				usage = chunk.Usage
			}
			if chunk.ReasoningContent != "" {
				delta, _ := json.Marshal(map[string]string{"text": chunk.ReasoningContent})
				emit("reasoning", string(delta))
			}
			if chunk.Content == "" {
				continue
			}

			full.WriteString(chunk.Content)
			delta, _ := json.Marshal(map[string]string{"text": chunk.Content})
			emit("delta", string(delta))
		}
	}
}

func writeReplaySSE(ctx context.Context, w http.ResponseWriter, session *service.StreamSession, lastEventID int) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		return
	}

	for _, ev := range session.EventsAfter(lastEventID) {
		fmt.Fprintf(w, "id: %d\n", ev.ID)
		fmt.Fprintf(w, "event: %s\n", ev.Event)
		fmt.Fprintf(w, "data: %s\n\n", ev.Data)
		flusher.Flush()
	}

	if session.IsDone() {
		return
	}

	sub, unsub := session.Subscribe()
	defer unsub()

	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-sub:
			if !ok {
				return
			}
			fmt.Fprintf(w, "id: %d\n", ev.ID)
			fmt.Fprintf(w, "event: %s\n", ev.Event)
			fmt.Fprintf(w, "data: %s\n\n", ev.Data)
			flusher.Flush()
			if ev.Event == "end" {
				return
			}
		}
	}
}

func parseLastEventID(r *http.Request) int {
	if v := r.Header.Get("Last-Event-ID"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	if v := r.URL.Query().Get("last_event_id"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return 0
}
