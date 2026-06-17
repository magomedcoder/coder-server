package delivery

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/magomedcoder/coder-server/internal/domain"
	"github.com/magomedcoder/coder-server/internal/service"
)

func (h *Handler) handleQueueStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}

	var stats domain.QueueStatsResponse
	if h.jobs != nil {
		stats = h.jobs.Stats(h.llm.RequestQueue())
	} else if h.llm != nil && h.llm.RequestQueue() != nil {
		q := h.llm.RequestQueue()
		stats.InFlight = q.InFlight()
		stats.Capacity = q.Capacity()
	}

	writeJSON(w, http.StatusOK, stats)
}

func (h *Handler) handleQueueJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}

	if h.jobs == nil {
		writeJSON(w, http.StatusNotFound, domain.NewErrorResponse("not_found", "persistent queue отключена"))
		return
	}

	id := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/v1/queue/jobs/"))
	if id == "" {
		writeBadRequest(w, "job id обязателен")
		return
	}

	job, ok := h.jobs.Get(id)
	if !ok {
		writeJSON(w, http.StatusNotFound, domain.NewErrorResponse("not_found", "job не найден"))
		return
	}

	writeJSON(w, http.StatusOK, h.jobs.ToStatus(job))
}

func (h *Handler) tryEnqueueOnOverload(w http.ResponseWriter, kind, requestID string, body any) bool {
	if h.jobs == nil {
		return false
	}

	jobID, err := h.jobs.Enqueue(kind, requestID, body)
	if err != nil {
		logReq(requestID, "очередь: не удалось поставить задачу kind=%s: %v", kind, err)
		writeJSON(w, http.StatusServiceUnavailable, domain.NewErrorResponse("overloaded", err.Error()))
		return true
	}

	logReq(requestID, "очередь: задача %s kind=%s", jobID, kind)

	writeJSON(w, http.StatusAccepted, domain.QueueJobStatus{
		JobID:     jobID,
		Status:    "pending",
		Kind:      kind,
		RequestID: requestID,
	})

	return true
}

func (h *Handler) handleAgentRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}

	if h.sandbox == nil {
		writeJSON(w, http.StatusServiceUnavailable, domain.NewErrorResponse("service_unavailable", "sandbox не настроен"))
		return
	}

	var req domain.AgentRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "некорректное тело JSON")
		return
	}

	if p := h.agentPolicy(); p != nil {
		if reason := p.ValidateRunCommand(req.Command); reason != "" {
			writeJSON(w, http.StatusForbidden, domain.NewErrorResponse("forbidden", reason))
			return
		}
	}

	stdout, stderr, exitCode, err := h.sandbox.Run(r.Context(), req.Command, req.Cwd)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, domain.NewErrorResponse("bad_request", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, domain.AgentRunResponse{
		Stdout:   stdout,
		Stderr:   stderr,
		ExitCode: exitCode,
	})
}

func (h *Handler) mapRunnerErrorWithQueue(w http.ResponseWriter, err error, kind, requestID string, body any) {
	if errors.Is(err, service.ErrQueueTimeout) && h.tryEnqueueOnOverload(w, kind, requestID, body) {
		return
	}

	h.mapRunnerError(w, err)
}
