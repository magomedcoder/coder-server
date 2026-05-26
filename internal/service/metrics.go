package service

import (
	"sync/atomic"
)

type Metrics struct {
	ChatRequests     atomic.Int64
	ChatErrors       atomic.Int64
	AgentSteps       atomic.Int64
	AgentErrors      atomic.Int64
	TokensPrompt     atomic.Int64
	TokensCompletion atomic.Int64
	LatencyTotalMs   atomic.Int64
	RequestCount     atomic.Int64
}

func NewMetrics() *Metrics {
	return &Metrics{}
}

type MetricsSnapshot struct {
	ChatRequests     int64   `json:"chat_requests"`
	ChatErrors       int64   `json:"chat_errors"`
	AgentSteps       int64   `json:"agent_steps"`
	AgentErrors      int64   `json:"agent_errors"`
	TokensPrompt     int64   `json:"tokens_prompt"`
	TokensCompletion int64   `json:"tokens_completion"`
	QueueInFlight    int     `json:"queue_in_flight"`
	QueueCapacity    int     `json:"queue_capacity"`
	ActiveStreams    int64   `json:"active_streams"`
	AvgLatencyMs     float64 `json:"avg_latency_ms"`
	ErrorRate        float64 `json:"error_rate"`
	QuotaUsedToday   int64   `json:"quota_tokens_used_today,omitempty"`
	QuotaMaxPerDay   int64   `json:"quota_tokens_max_per_day,omitempty"`
}

func (m *Metrics) Snapshot(queue *RequestQueue, activeStreams int64, quota *TokenQuota) MetricsSnapshot {
	if m == nil {
		return MetricsSnapshot{}
	}

	reqs := m.RequestCount.Load()
	latency := m.LatencyTotalMs.Load()
	errs := m.ChatErrors.Load() + m.AgentErrors.Load()

	snap := MetricsSnapshot{
		ChatRequests:     m.ChatRequests.Load(),
		ChatErrors:       m.ChatErrors.Load(),
		AgentSteps:       m.AgentSteps.Load(),
		AgentErrors:      m.AgentErrors.Load(),
		TokensPrompt:     m.TokensPrompt.Load(),
		TokensCompletion: m.TokensCompletion.Load(),
		ActiveStreams:    activeStreams,
	}
	if queue != nil {
		snap.QueueInFlight = queue.InFlight()
		snap.QueueCapacity = queue.Capacity()
	}

	if reqs > 0 {
		snap.AvgLatencyMs = float64(latency) / float64(reqs)
		snap.ErrorRate = float64(errs) / float64(reqs)
	}

	if quota != nil {
		snap.QuotaUsedToday, snap.QuotaMaxPerDay = quota.Snapshot()
	}

	return snap
}

func (m *Metrics) RecordRequest(durationMs int64) {
	if m == nil {
		return
	}
	m.RequestCount.Add(1)
	m.LatencyTotalMs.Add(durationMs)
}

func (m *Metrics) RecordTokens(prompt, completion int32) {
	if m == nil {
		return
	}

	if prompt > 0 {
		m.TokensPrompt.Add(int64(prompt))
	}

	if completion > 0 {
		m.TokensCompletion.Add(int64(completion))
	}
}
