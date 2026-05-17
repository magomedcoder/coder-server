package llmrunner

import (
	"math"
	"sort"
	"strings"
)

type RunnerCoreHints struct {
	MaxContextTokens           int
	LLMHistoryMaxMessages      int
	LLMHistorySummarizeDropped bool
	SummaryMaxInputRunes       int
	SummaryModel               string
	SummaryRunnerListenAddress string
	SummaryCacheEntries        int
	MaxToolInvocationRounds    int
}

func DefaultRunnerCoreHints() RunnerCoreHints {
	return RunnerCoreHints{}
}

func normHistMax(n int) int {
	if n <= 0 {
		return 0
	}
	return n
}

func normApproxTok(n int) int {
	if n <= 0 {
		return 0
	}

	if n < 512 {
		return 512
	}

	if n > 500_000 {
		return 500_000
	}

	return n
}

func normSummaryRunes(n int) int {
	if n <= 0 {
		return 0
	}
	return n
}

func normToolRounds(n int) int {
	if n <= 0 {
		return 0
	}

	return n
}

func normCacheEntries(n int) int {
	if n < 0 {
		return 0
	}

	if n > 50_000 {
		return 50_000
	}

	return n
}

func FinalizeChatHints(h RunnerCoreHints) RunnerCoreHints {
	h.LLMHistoryMaxMessages = normHistMax(h.LLMHistoryMaxMessages)
	h.MaxContextTokens = normApproxTok(h.MaxContextTokens)
	h.SummaryMaxInputRunes = normSummaryRunes(h.SummaryMaxInputRunes)
	h.MaxToolInvocationRounds = normToolRounds(h.MaxToolInvocationRounds)
	h.SummaryCacheEntries = normCacheEntries(h.SummaryCacheEntries)
	h.SummaryModel = strings.TrimSpace(h.SummaryModel)
	h.SummaryRunnerListenAddress = strings.TrimSpace(h.SummaryRunnerListenAddress)

	return h
}

func (r *Registry) AggregateChatHints() RunnerCoreHints {
	type pair struct {
		addr string
		h    RunnerCoreHints
	}

	r.mu.RLock()
	pairs := make([]pair, 0, len(r.runners))
	for addr, st := range r.runners {
		if !st.Enabled || st.Hints == nil {
			continue
		}
		pairs = append(pairs, pair{addr, *st.Hints})
	}
	r.mu.RUnlock()

	if len(pairs) == 0 {
		return FinalizeChatHints(DefaultRunnerCoreHints())
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].addr < pairs[j].addr
	})

	minPositive := func(get func(RunnerCoreHints) int) int {
		v := math.MaxInt32
		any := false
		for _, p := range pairs {
			n := get(p.h)
			if n <= 0 {
				continue
			}
			any = true
			if n < v {
				v = n
			}
		}
		if !any {
			return 0
		}
		return v
	}

	out := RunnerCoreHints{}
	out.MaxContextTokens = minPositive(func(h RunnerCoreHints) int {
		return h.MaxContextTokens
	})

	msgMin := minPositive(func(h RunnerCoreHints) int {
		return h.LLMHistoryMaxMessages
	})
	if msgMin > 0 {
		out.LLMHistoryMaxMessages = msgMin
	}

	trMin := minPositive(func(h RunnerCoreHints) int {
		return h.MaxToolInvocationRounds
	})
	if trMin > 0 {
		out.MaxToolInvocationRounds = trMin
	}

	for _, p := range pairs {
		if p.h.LLMHistorySummarizeDropped {
			out.LLMHistorySummarizeDropped = true
			break
		}
	}

	maxCache := 0
	for _, p := range pairs {
		if p.h.SummaryCacheEntries > maxCache {
			maxCache = p.h.SummaryCacheEntries
		}
	}
	out.SummaryCacheEntries = maxCache

	summaryRunes := math.MaxInt32
	for _, p := range pairs {
		if p.h.SummaryMaxInputRunes > 0 && p.h.SummaryMaxInputRunes < summaryRunes {
			summaryRunes = p.h.SummaryMaxInputRunes
		}
	}
	if summaryRunes != math.MaxInt32 {
		out.SummaryMaxInputRunes = summaryRunes
	}

	bottleneckCtx := math.MaxInt32
	bottleneckIdx := -1
	for i, p := range pairs {
		if p.h.MaxContextTokens <= 0 {
			continue
		}

		if p.h.MaxContextTokens < bottleneckCtx {
			bottleneckCtx = p.h.MaxContextTokens
			bottleneckIdx = i
		}
	}

	if bottleneckIdx >= 0 {
		bh := pairs[bottleneckIdx].h
		out.SummaryModel = strings.TrimSpace(bh.SummaryModel)
		out.SummaryRunnerListenAddress = strings.TrimSpace(bh.SummaryRunnerListenAddress)
	}

	if out.SummaryModel == "" {
		for _, p := range pairs {
			if s := strings.TrimSpace(p.h.SummaryModel); s != "" {
				out.SummaryModel = s
				break
			}
		}
	}

	if out.SummaryRunnerListenAddress == "" {
		for _, p := range pairs {
			if s := strings.TrimSpace(p.h.SummaryRunnerListenAddress); s != "" {
				out.SummaryRunnerListenAddress = s
				break
			}
		}
	}

	return FinalizeChatHints(out)
}
