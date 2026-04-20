package mcpsafe

import (
	"log"
	"net/http"
	"runtime/debug"
	"strings"
)

func RecoverPanic(origin string, next http.Handler) http.Handler {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		origin = "mcp-http"
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("%s: HTTP panic recovered: %v\n%s", origin, rec, debug.Stack())
			}
		}()
		next.ServeHTTP(w, r)
	})
}
