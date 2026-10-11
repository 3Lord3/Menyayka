package main

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func quietLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return logger
}

// A panic in a background tick must be contained; one bad tick cannot take the API down.
func TestRunTickRecoversFromPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("runTick leaked a panic: %v", r)
		}
	}()
	runTick(context.Background(), quietLogger(), "test", func(context.Context) error {
		panic("boom")
	}, time.Second)
}

func TestRunTickInvokesFunctionOnce(t *testing.T) {
	calls := 0
	runTick(context.Background(), quietLogger(), "test", func(context.Context) error {
		calls++
		return errors.New("tick failed")
	}, time.Second)
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

// Each tick gets its own deadline so a stalled query cannot delay the next ones.
func TestRunTickAppliesTimeout(t *testing.T) {
	start := time.Now()
	runTick(context.Background(), quietLogger(), "test", func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}, 50*time.Millisecond)

	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("runTick took %s, want a bounded tick", elapsed)
	}
}

// Cancelling the parent context must stop the worker promptly.
func TestRunExpiryWorkerStopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		runExpiryWorker(ctx, noopExpirer{}, quietLogger())
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runExpiryWorker did not stop after context cancel")
	}
}

type noopExpirer struct{}

func (noopExpirer) ExpireDue(context.Context) error { return nil }

// A worker stuck forever must not block shutdown.
func TestWaitTimeoutGivesUp(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1) // never Done()

	start := time.Now()
	waitTimeout(&wg, 50*time.Millisecond)
	if elapsed := time.Since(start); elapsed < 50*time.Millisecond || elapsed > 2*time.Second {
		t.Fatalf("waitTimeout took %s, want ~50ms", elapsed)
	}
}

func TestWaitTimeoutReturnsWhenWorkCompletes(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond)
	}()

	done := make(chan struct{})
	go func() {
		waitTimeout(&wg, time.Minute)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("waitTimeout did not return after the work finished")
	}
}
