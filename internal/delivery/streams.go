package delivery

import "sync/atomic"

type ActiveStreams struct {
	count atomic.Int64
}

func NewActiveStreamsTracker() *ActiveStreams {
	return &ActiveStreams{}
}

func (a *ActiveStreams) Inc() {
	a.count.Add(1)
}

func (a *ActiveStreams) Dec() {
	a.count.Add(-1)
}

func (a *ActiveStreams) Count() int64 {
	return a.count.Load()
}
