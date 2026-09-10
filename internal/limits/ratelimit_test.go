package limits

import (
	"testing"
	"time"
)

func TestLimiter_BurstAndDeny(t *testing.T) {
	l := NewLimiter(60, 3) // 1/sec, burst 3
	fixed := time.Unix(1_700_000_000, 0)
	l.now = func() time.Time { return fixed }

	for i := 0; i < 3; i++ {
		if !l.Allow("1") {
			t.Fatalf("burst slot %d should allow", i)
		}
	}
	if l.Allow("1") {
		t.Fatal("should deny after burst")
	}

	// Advance enough for one token
	fixed = fixed.Add(time.Second)
	if !l.Allow("1") {
		t.Fatal("should allow after refill")
	}
}

func TestLimiter_PerKey(t *testing.T) {
	l := NewLimiter(30, 1)
	if !l.Allow("1") || !l.Allow("2") {
		t.Fatal("different keys should have independent buckets")
	}
	if l.Allow("1") {
		t.Fatal("user 1 burst exhausted")
	}
	if l.Allow("2") {
		t.Fatal("user 2 burst exhausted")
	}
}
