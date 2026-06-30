package mcpclient

import (
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	sessionRootsMu sync.RWMutex
	sessionRoots   []*mcp.Root
)

func SetSessionRoots(roots []*mcp.Root) {
	sessionRootsMu.Lock()
	defer sessionRootsMu.Unlock()
	if len(roots) == 0 {
		sessionRoots = nil
		return
	}
	sessionRoots = make([]*mcp.Root, len(roots))
	copy(sessionRoots, roots)
}

func rootsForSession() []*mcp.Root {
	sessionRootsMu.RLock()
	defer sessionRootsMu.RUnlock()
	if len(sessionRoots) == 0 {
		return nil
	}
	out := make([]*mcp.Root, len(sessionRoots))
	copy(out, sessionRoots)
	return out
}

func RootsFromConfigStrings(rows []string) ([]*mcp.Root, error) {
	var out []*mcp.Root
	for i, raw := range rows {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		uri, err := normalizeFileRootURI(raw)
		if err != nil {
			return nil, fmt.Errorf("mcp.roots[%d] %q: %w", i, raw, err)
		}
		out = append(out, &mcp.Root{
			URI:  uri,
			Name: fmt.Sprintf("cfg-%d", i),
		})
	}
	return out, nil
}

func normalizeFileRootURI(s string) (string, error) {
	low := strings.ToLower(s)
	if strings.HasPrefix(low, "file://") {
		u, err := url.Parse(s)
		if err != nil {
			return "", err
		}
		if u.Scheme != "file" {
			return "", fmt.Errorf("ожидается file://, получено %q", u.Scheme)
		}
		return u.String(), nil
	}

	abs, err := filepath.Abs(s)
	if err != nil {
		return "", err
	}

	return fileURIFromLocalPath(abs), nil
}

func fileURIFromLocalPath(abs string) string {
	abs = filepath.Clean(abs)
	slash := filepath.ToSlash(abs)
	if runtime.GOOS == "windows" {
		if len(slash) >= 2 && slash[1] == ':' {
			return "file:///" + slash
		}
		return "file:///" + strings.TrimPrefix(slash, "/")
	}
	return "file://" + slash
}
