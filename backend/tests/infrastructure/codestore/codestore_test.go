package codestore_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Avito-Team-Not-Found/menyayka/pkg/codestore"
)

func TestStore_SaveAndGet(t *testing.T) {
	s := codestore.New()
	s.Save("key", "value", time.Minute)

	v, ok := s.Get("key")
	if !ok || v != "value" {
		t.Fatalf("expected (value, true), got (%q, %v)", v, ok)
	}
}

func TestStore_GetMissing(t *testing.T) {
	s := codestore.New()

	if _, ok := s.Get("missing"); ok {
		t.Fatal("expected ok=false for missing key")
	}
}

func TestStore_ExpiresAfterTTL(t *testing.T) {
	s := codestore.New()
	s.Save("key", "value", 10*time.Millisecond)

	time.Sleep(30 * time.Millisecond)

	if _, ok := s.Get("key"); ok {
		t.Fatal("expected key to expire after TTL")
	}
}

func TestStore_SaveOverwritesPreviousValue(t *testing.T) {
	s := codestore.New()
	s.Save("key", "first", time.Minute)
	s.Save("key", "second", time.Minute)

	v, ok := s.Get("key")
	if !ok || v != "second" {
		t.Fatalf("expected (second, true), got (%q, %v)", v, ok)
	}
}

func TestStore_Delete(t *testing.T) {
	s := codestore.New()
	s.Save("key", "value", time.Minute)
	s.Delete("key")

	if _, ok := s.Get("key"); ok {
		t.Fatal("expected key to be gone after Delete")
	}
}

func TestStore_ConsumeRemovesValue(t *testing.T) {
	s := codestore.New()
	s.Save("ticket", "user-1", time.Minute)

	v, ok := s.Consume("ticket")
	if !ok || v != "user-1" {
		t.Fatalf("first Consume = (%q, %v), want (user-1, true)", v, ok)
	}
	if _, ok := s.Consume("ticket"); ok {
		t.Fatal("second Consume must fail: ticket is one-time")
	}
}

func TestStore_ConsumeExpired(t *testing.T) {
	s := codestore.New()
	s.Save("ticket", "user-1", 10*time.Millisecond)
	time.Sleep(30 * time.Millisecond)

	if _, ok := s.Consume("ticket"); ok {
		t.Fatal("expired key must not be consumed")
	}
}

func TestStore_ConsumeIsAtomicUnderRace(t *testing.T) {
	s := codestore.New()
	s.Save("ticket", "user-1", time.Minute)

	const goroutines = 32
	var wg sync.WaitGroup
	var successes int64
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			if _, ok := s.Consume("ticket"); ok {
				atomic.AddInt64(&successes, 1)
			}
		}()
	}
	wg.Wait()
	if successes != 1 {
		t.Fatalf("Consume succeeded %d times, want exactly 1", successes)
	}
}

func TestStore_SweepRemovesOnlyExpired(t *testing.T) {
	s := codestore.New()
	s.Save("expired", "a", time.Millisecond)
	s.Save("fresh", "b", time.Hour)
	time.Sleep(20 * time.Millisecond)

	if removed := s.Sweep(time.Now()); removed != 1 {
		t.Fatalf("Sweep removed %d, want 1", removed)
	}
	if _, ok := s.Get("fresh"); !ok {
		t.Fatal("Sweep must not remove live entries")
	}
	if s.Len() != 1 {
		t.Fatalf("Len = %d, want 1", s.Len())
	}
}
