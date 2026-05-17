package sessionupload

import (
	"fmt"
	"sync"
	"time"
)

const (
	PutSessionFileRateWindow       = time.Minute
	PutSessionFileMaxBytesPerMin   = 48 * 1024 * 1024
	PutSessionFileMaxUploadsPerMin = 80
	putSessionFileRatePruneEntries = 4096
)

type Limiter struct {
	mu      sync.Mutex
	perUser map[int]*uploadRollingWindow
	now     func() time.Time
}

func NewLimiter() *Limiter {
	return &Limiter{perUser: make(map[int]*uploadRollingWindow)}
}

func NewLimiterWithClock(now func() time.Time) *Limiter {
	return &Limiter{perUser: make(map[int]*uploadRollingWindow), now: now}
}

type uploadRollingWindow struct {
	windowStart time.Time
	bytes       int64
	n           int
}

func (l *Limiter) currentTime() time.Time {
	if l.now != nil {
		return l.now()
	}

	return time.Now()
}

func (l *Limiter) CheckPutSessionFile(userID int, byteLen int) error {
	if byteLen < 0 {
		byteLen = 0
	}

	b := int64(byteLen)
	now := l.currentTime()

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.perUser == nil {
		l.perUser = make(map[int]*uploadRollingWindow)
	}

	w := l.perUser[userID]
	if w == nil || now.Sub(w.windowStart) >= PutSessionFileRateWindow {
		l.perUser[userID] = &uploadRollingWindow{
			windowStart: now,
			bytes:       b,
			n:           1,
		}

		l.maybePruneLocked(now)

		return nil
	}

	if w.n >= PutSessionFileMaxUploadsPerMin {
		return fmt.Errorf("слишком много загрузок файлов за минуту (лимит %d)", PutSessionFileMaxUploadsPerMin)
	}

	if w.bytes+b > PutSessionFileMaxBytesPerMin {
		return fmt.Errorf(
			"превышен объём загрузок за минуту: лимит %d МиБ",
			PutSessionFileMaxBytesPerMin/(1024*1024),
		)
	}

	w.bytes += b
	w.n++
	l.maybePruneLocked(now)
	return nil
}

func (l *Limiter) maybePruneLocked(now time.Time) {
	if len(l.perUser) < putSessionFileRatePruneEntries {
		return
	}

	for uid, w := range l.perUser {
		if w == nil || now.Sub(w.windowStart) >= PutSessionFileRateWindow {
			delete(l.perUser, uid)
		}
	}
}
