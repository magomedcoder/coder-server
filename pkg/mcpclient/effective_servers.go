package mcpclient

import (
	"slices"
)

func EffectiveServerIDs(explicit, fallbackIDs []int64) []int64 {
	var source []int64
	if len(explicit) > 0 {
		source = explicit
	} else {
		source = fallbackIDs
	}

	if len(source) == 0 {
		return nil
	}

	out := make([]int64, 0, len(source))
	seen := map[int64]struct{}{}
	for _, sid := range source {
		if sid <= 0 {
			continue
		}
		if _, ok := seen[sid]; ok {
			continue
		}
		seen[sid] = struct{}{}
		out = append(out, sid)
	}

	slices.Sort(out)
	return out
}
