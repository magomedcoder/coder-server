package quota

import (
	"sync"
	"time"
)

type Quota struct {
	mu        sync.Mutex
	maxPerDay int64
	usedToday int64
	dayStart  time.Time
}

func New(maxPerDay int64) *Quota {
	if maxPerDay <= 0 {
		return nil
	}

	return &Quota{
		maxPerDay: maxPerDay,
		dayStart:  startOfDay(time.Now()),
	}
}

func (q *Quota) WouldAllow(add int64) bool {
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

func (q *Quota) Record(add int64) {
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

func (q *Quota) Allow(add int64) bool {
	if !q.WouldAllow(add) {
		return false
	}
	q.Record(add)

	return true
}

func (q *Quota) Snapshot() (used, max int64) {
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
