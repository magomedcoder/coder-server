package runnerresolve

import (
	"context"
	"strings"
	"time"

	"github.com/magomedcoder/gen/pkg/llmrunner"
)

const DefaultProbeTimeout = 5 * time.Second

func ListEnabledEntries(ctx context.Context, reg *llmrunner.Registry, pool *llmrunner.Pool) []Entry {
	if reg == nil {
		return nil
	}

	runners := reg.ListRunners()
	out := make([]Entry, 0, len(runners))
	for _, r := range runners {
		if !r.Enabled {
			continue
		}
		addr := strings.TrimSpace(r.Address)
		if addr == "" {
			continue
		}

		entry := Entry{
			ID:            r.ID,
			Address:       addr,
			Name:          strings.TrimSpace(r.Name),
			SelectedModel: strings.TrimSpace(r.SelectedModel),
			Enabled:       true,
		}
		if pool != nil {
			probeCtx, cancel := context.WithTimeout(ctx, DefaultProbeTimeout)
			pr := pool.ProbeLLMRunner(probeCtx, addr)
			cancel()
			entry.Probed = true
			entry.Connected = pr.Connected
		}
		out = append(out, entry)
	}

	return out
}
