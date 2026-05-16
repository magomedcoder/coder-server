package delivery

import (
	"fmt"
	"net/http"

	gendomain "github.com/magomedcoder/gen/pkg/domain"
)

func writeRunnerSSE(w http.ResponseWriter, chunks <-chan gendomain.LLMStreamChunk) {
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

	for chunk := range chunks {
		if chunk.Content == "" {
			continue
		}

		fmt.Fprintf(w, "event: delta\n")
		fmt.Fprintf(w, "data: {\"text\":%q}\n\n", chunk.Content)
		flusher.Flush()
	}

	fmt.Fprintf(w, "event: end\n")
	fmt.Fprintf(w, "data: {\"finish\":\"stop\"}\n\n")
	flusher.Flush()
}
