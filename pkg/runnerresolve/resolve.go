package runnerresolve

import (
	"sort"
	"strings"
)

type Entry struct {
	ID            int64
	Address       string
	Name          string
	SelectedModel string
	Enabled       bool
	Connected     bool
	Probed        bool
}

type Selection struct {
	Address             string
	Name                string
	Enabled             bool
	Connected           bool
	UnknownConnectivity bool
}

func AvailableAddresses(entries []Entry) []string {
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

func RunnerNames(entries []Entry) map[string]string {
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

func HealthFor(address string, entries []Entry) (enabled bool, connected bool, probed bool, ok bool) {
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

func IsRunnable(address string, entries []Entry) bool {
	enabled, connected, probed, ok := HealthFor(address, entries)
	if !ok || !enabled {
		return false
	}

	if probed && !connected {
		return false
	}

	return true
}

func Resolve(available []string, entries []Entry, candidate string) Selection {
	if len(available) == 0 {
		return Selection{}
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
		if !contains(available, addr) {
			continue
		}

		if IsRunnable(addr, entries) {
			return selectionFor(addr, entries)
		}
	}

	return selectionFor(available[0], entries)
}

func selectionFor(address string, entries []Entry) Selection {
	enabled, connected, probed, ok := HealthFor(address, entries)
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
		return Selection{
			Address: address,
			Name:    name,
		}
	}

	return Selection{
		Address:             address,
		Name:                name,
		Enabled:             enabled,
		Connected:           connected,
		UnknownConnectivity: !probed,
	}
}

func contains(list []string, item string) bool {
	for _, s := range list {
		if s == item {
			return true
		}
	}
	return false
}
