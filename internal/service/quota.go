package service

import (
	"sync"
	"time"
)

type TokenQuota struct {
	mu        sync.Mutex
	maxPerDay int64
	usedToday int64
	dayStart  time.Time
}

func NewTokenQuota(maxPerDay int64) *TokenQuota {
	if maxPerDay <= 0 {
		return nil
	}

	return &TokenQuota{
		maxPerDay: maxPerDay,
		dayStart:  startOfDay(time.Now()),
	}
}

func (q *TokenQuota) WouldAllow(add int64) bool {
	if q == nil || add <= 0 {
		return true
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now()
	if now.Sub(q.dayStart) >= 24*time.Hour {
		q.usedToday = 0
		q.dayStart = startOfDay(now)
	}

	return q.usedToday+add <= q.maxPerDay
}

func (q *TokenQuota) Record(add int64) {
	if q == nil || add <= 0 {
		return
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now()
	if now.Sub(q.dayStart) >= 24*time.Hour {
		q.usedToday = 0
		q.dayStart = startOfDay(now)
	}
	q.usedToday += add
}

func (q *TokenQuota) Allow(add int64) bool {
	if !q.WouldAllow(add) {
		return false
	}
	q.Record(add)

	return true
}

func (q *TokenQuota) Snapshot() (used, max int64) {
	if q == nil {
		return 0, 0
	}

	q.mu.Lock()
	defer q.mu.Unlock()
	return q.usedToday, q.maxPerDay
}

func startOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}
