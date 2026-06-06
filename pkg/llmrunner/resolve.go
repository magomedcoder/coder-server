package llmrunner

import (
	"context"
	"slices"
	"sort"
	"strings"
	"time"
)

const DefaultResolveProbeTimeout = 5 * time.Second

type ResolveEntry struct {
	ID            int64
	Address       string
	Name          string
	SelectedModel string
	Enabled       bool
	Connected     bool
	Probed        bool
}

type ResolveSelection struct {
	Address             string
	Name                string
	Enabled             bool
	Connected           bool
	UnknownConnectivity bool
}

func AvailableRunnerAddresses(entries []ResolveEntry) []string {
	set := make(map[string]struct{}, len(entries))
	for _, e := range entries {
		if !e.Enabled {
			continue
		}

		addr := strings.TrimSpace(e.Address)
		if addr == "" {
			continue
		}

		set[addr] = struct{}{}
	}

	out := make([]string, 0, len(set))
	for addr := range set {
		out = append(out, addr)
	}

	sort.Strings(out)

	return out
}

func RunnerDisplayNames(entries []ResolveEntry) map[string]string {
	names := make(map[string]string)
	for _, e := range entries {
		if !e.Enabled {
			continue
		}

		addr := strings.TrimSpace(e.Address)
		if addr == "" {
			continue
		}

		n := strings.TrimSpace(e.Name)
		if n == "" {
			n = addr
		}

		names[addr] = n
	}

	return names
}

func RunnerHealthFor(address string, entries []ResolveEntry) (enabled bool, connected bool, probed bool, ok bool) {
	address = strings.TrimSpace(address)
	if address == "" {
		return false, false, false, false
	}

	for _, e := range entries {
		if strings.TrimSpace(e.Address) != address {
			continue
		}

		if !e.Probed {
			return e.Enabled, false, false, true
		}

		return e.Enabled, e.Connected, true, true
	}

	return false, false, false, false
}

func RunnerIsRunnable(address string, entries []ResolveEntry) bool {
	enabled, connected, probed, ok := RunnerHealthFor(address, entries)
	if !ok || !enabled {
		return false
	}

	if probed && !connected {
		return false
	}

	return true
}

func ResolveRunner(available []string, entries []ResolveEntry, candidate string) ResolveSelection {
	if len(available) == 0 {
		return ResolveSelection{}
	}

	candidate = strings.TrimSpace(candidate)
	tryOrder := make([]string, 0, len(available)+1)
	if candidate != "" {
		tryOrder = append(tryOrder, candidate)
	}

	tryOrder = append(tryOrder, available...)

	seen := make(map[string]struct{}, len(tryOrder))
	for _, addr := range tryOrder {
		addr = strings.TrimSpace(addr)
		if addr == "" {
			continue
		}

		if _, dup := seen[addr]; dup {
			continue
		}

		seen[addr] = struct{}{}
		if !containsString(available, addr) {
			continue
		}

		if RunnerIsRunnable(addr, entries) {
			return selectionForRunner(addr, entries)
		}
	}

	return selectionForRunner(available[0], entries)
}

func ListEnabledRunnerEntries(ctx context.Context, reg *Registry, pool *Pool) []ResolveEntry {
	if reg == nil {
		return nil
	}

	runners := reg.ListRunners()
	out := make([]ResolveEntry, 0, len(runners))
	for _, r := range runners {
		if !r.Enabled {
			continue
		}
		addr := strings.TrimSpace(r.Address)
		if addr == "" {
			continue
		}

		entry := ResolveEntry{
			ID:            r.ID,
			Address:       addr,
			Name:          strings.TrimSpace(r.Name),
			SelectedModel: strings.TrimSpace(r.SelectedModel),
			Enabled:       true,
		}
		if pool != nil {
			probeCtx, cancel := context.WithTimeout(ctx, DefaultResolveProbeTimeout)
			pr := pool.ProbeLLMRunner(probeCtx, addr)
			cancel()
			entry.Probed = true
			entry.Connected = pr.Connected
		}
		out = append(out, entry)
	}

	return out
}

func selectionForRunner(address string, entries []ResolveEntry) ResolveSelection {
	enabled, connected, probed, ok := RunnerHealthFor(address, entries)
	name := address
	for _, e := range entries {
		if strings.TrimSpace(e.Address) == address {
			if n := strings.TrimSpace(e.Name); n != "" {
				name = n
			}
			break
		}
	}

	if !ok {
		return ResolveSelection{
			Address: address,
			Name:    name,
		}
	}

	return ResolveSelection{
		Address:             address,
		Name:                name,
		Enabled:             enabled,
		Connected:           connected,
		UnknownConnectivity: !probed,
	}
}

func containsString(list []string, item string) bool {
	return slices.Contains(list, item)
}
