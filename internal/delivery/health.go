package delivery

import (
	"net/http"

	"github.com/magomedcoder/coder-server/internal/domain"
)

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	h.writeHealthResponse(w, r, true)
}

func (h *Handler) handleHealthLive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) handleHealthReady(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	h.writeHealthResponse(w, r, false)
}

func (h *Handler) writeHealthResponse(w http.ResponseWriter, r *http.Request, includeCapabilities bool) {
	resp := domain.HealthResponse{
		OK: false,
	}
	ok, err := h.llm.CheckConnection(r.Context())
	resp.Runner = &domain.HealthRunnerInfo{Connected: ok && err == nil}

	if ok && err == nil {
		probe, addr, probeErr := h.llm.ProbeBestRunner(r.Context())
		if probeErr == nil {
			resp.Runner.Address = addr
			if probe.LoadedModel != nil {
				resp.Runner.ModelLoaded = probe.LoadedModel.Loaded
				if probe.LoadedModel.DisplayName != "" {
					resp.Runner.Model = probe.LoadedModel.DisplayName
				} else {
					resp.Runner.Model = probe.LoadedModel.GGUFBasename
				}
			}
			resp.OK = probe.Connected && resp.Runner.ModelLoaded
		}
	}

	if includeCapabilities {
		hints := h.llm.ChatHints()
		caps := &domain.ModelCapabilities{
			JSONMode: true,
			Tools:    true,
		}
		if hints.MaxContextTokens > 0 {
			caps.MaxContextTokens = hints.MaxContextTokens
		} else if budget := h.cfg.EffectiveContextTokenBudget(0); budget > 0 {
			caps.MaxContextTokens = budget
		}
		if _, err := h.llm.Embed(r.Context(), "ping"); err == nil {
			caps.Embeddings = true
		}
		if h.mcp != nil && h.mcp.Enabled() {
			caps.MCPServers = h.mcp.ServerCount()
		}
		resp.Capabilities = caps
	}

	status := http.StatusOK
	if !resp.OK {
		status = http.StatusServiceUnavailable
	}
	writeJSON(w, status, resp)
}
