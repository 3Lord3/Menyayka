package realtime

import (
	"testing"
	"time"
)

func TestRateLimiterAllowsUpToMax(t *testing.T) {
	limiter := newRateLimiter(2, time.Minute)
	if !limiter.allow() || !limiter.allow() {
		t.Fatal("first two messages must pass")
	}
	if limiter.allow() {
		t.Fatal("third message over the limit must be rejected")
	}
}

func TestRateLimiterClosesOnlyAfterPersistentFlooding(t *testing.T) {
	limiter := newRateLimiter(1, time.Minute)
	if !limiter.allow() {
		t.Fatal("first message must pass")
	}
	for i := 0; i < 6; i++ {
		if limiter.allow() {
			t.Fatalf("message %d over the limit = true, want false", i)
		}
	}
	if !limiter.exceeded() {
		t.Fatal("a persistent flood must eventually trip exceeded()")
	}
}

// One violation is a warning, not a ban.
func TestRateLimiterSingleViolationDoesNotTrip(t *testing.T) {
	limiter := newRateLimiter(1, time.Minute)
	_ = limiter.allow()
	_ = limiter.allow()
	if limiter.exceeded() {
		t.Fatal("a single violation must not close the connection")
	}
}

func TestRateLimiterDisabledWhenMaxIsZero(t *testing.T) {
	limiter := newRateLimiter(0, time.Minute)
	for i := 0; i < 100; i++ {
		if !limiter.allow() {
			t.Fatal("max=0 must disable the limit")
		}
	}
}
