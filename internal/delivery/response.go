package delivery

import (
	"encoding/json"
	"net/http"

	"github.com/magomedcoder/coder-server/internal/domain"
)

func writeMethodNotAllowed(w http.ResponseWriter, expected string) {
	writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
		"error": map[string]any{
			"code":    "method_not_allowed",
			"message": "ожидается метод " + expected,
		},
	})
}

func writeBadRequest(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusBadRequest, domain.NewErrorResponse("bad_request", message))
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
