package limits

import (
	"sync"
	"time"
)

// Limiter is a per-key token bucket with a bounded map and TTL GC.
type Limiter struct {
	mu       sync.Mutex
	rate     float64 // tokens per second
	burst    float64
	entries  map[int64]*bucket
	maxKeys  int
	idleTTL  time.Duration
	now      func() time.Time
}

type bucket struct {
	tokens float64
	last   time.Time
}

// NewLimiter creates a per-user limiter. ratePerMin is refill rate; burst is max tokens.
func NewLimiter(ratePerMin, burst int) *Limiter {
	if ratePerMin <= 0 {
		ratePerMin = 1
	}
	if burst <= 0 {
		burst = 1
	}
	return &Limiter{
		rate:    float64(ratePerMin) / 60.0,
		burst:   float64(burst),
		entries: make(map[int64]*bucket),
		maxKeys: 10_000,
		idleTTL: 10 * time.Minute,
		now:     time.Now,
	}
}

// Allow reports whether key may proceed (consumes one token on success).
func (l *Limiter) Allow(key int64) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.gcLocked(now)

	b, ok := l.entries[key]
	if !ok {
		if len(l.entries) >= l.maxKeys {
			l.evictOneLocked()
		}
		l.entries[key] = &bucket{tokens: l.burst - 1, last: now}
		return true
	}

	elapsed := now.Sub(b.last).Seconds()
	if elapsed > 0 {
		b.tokens += elapsed * l.rate
		if b.tokens > l.burst {
			b.tokens = l.burst
		}
		b.last = now
	}
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (l *Limiter) gcLocked(now time.Time) {
	for k, b := range l.entries {
		if now.Sub(b.last) > l.idleTTL {
			delete(l.entries, k)
		}
	}
}

func (l *Limiter) evictOneLocked() {
	var oldest int64
	var oldestTime time.Time
	first := true
	for k, b := range l.entries {
		if first || b.last.Before(oldestTime) {
			oldest = k
			oldestTime = b.last
			first = false
		}
	}
	if !first {
		delete(l.entries, oldest)
	}
}

// LenForTest returns tracked keys (tests only).
func (l *Limiter) LenForTest() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.entries)
}
