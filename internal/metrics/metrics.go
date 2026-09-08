// Package metrics provides process-local counters without PII or handwriting content.
package metrics

import (
	"sync"
	"sync/atomic"
)

var (
	mu      sync.Mutex
	counters = map[string]*atomic.Int64{}
)

// Add increments a named counter by delta (usually 1).
func Add(name string, delta int64) {
	if name == "" || delta == 0 {
		return
	}
	mu.Lock()
	c, ok := counters[name]
	if !ok {
		c = &atomic.Int64{}
		counters[name] = c
	}
	mu.Unlock()
	c.Add(delta)
}

// Get returns the current value of a counter.
func Get(name string) int64 {
	mu.Lock()
	c := counters[name]
	mu.Unlock()
	if c == nil {
		return 0
	}
	return c.Load()
}

// ResetForTest clears all counters (tests only).
func ResetForTest() {
	mu.Lock()
	counters = map[string]*atomic.Int64{}
	mu.Unlock()
}

// SnapshotForTest returns a copy of all counters (tests only).
func SnapshotForTest() map[string]int64 {
	mu.Lock()
	defer mu.Unlock()
	out := make(map[string]int64, len(counters))
	for k, c := range counters {
		out[k] = c.Load()
	}
	return out
}
