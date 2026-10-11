package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/Avito-Team-Not-Found/menyayka/internal/core/database"
)

type fakeTM struct {
	commitFailure bool
}

func (f *fakeTM) WithinTransaction(ctx context.Context, fn func(context.Context, database.Tx) error) error {
	if err := fn(ctx, nil); err != nil {
		return err
	}
	if f.commitFailure {
		return context.DeadlineExceeded
	}
	return nil
}

type recorded struct {
	rooms []string
	msg   []byte
}

type fakeBroadcaster struct {
	mu      sync.Mutex
	records []recorded
}

func (f *fakeBroadcaster) PublishRooms(rooms []string, msg []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.records = append(f.records, recorded{rooms: rooms, msg: msg})
}

func (f *fakeBroadcaster) countRoom(room string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, r := range f.records {
		for _, got := range r.rooms {
			if got == room {
				n++
				break
			}
		}
	}
	return n
}

func (f *fakeBroadcaster) eventTypes() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	types := make([]string, 0, len(f.records))
	for _, r := range f.records {
		var event message
		_ = json.Unmarshal(r.msg, &event)
		types = append(types, event.Type)
	}
	return types
}

type fakeResolver struct {
	mu    sync.Mutex
	calls map[int64]int
	users map[int64][]string
}

func newFakeResolver() *fakeResolver {
	return &fakeResolver{calls: map[int64]int{}, users: map[int64][]string{}}
}

func (f *fakeResolver) ListChainParticipantUserIDs(_ context.Context, chainID int64) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls[chainID]++
	return f.users[chainID], nil
}

func TestTxEmitterFlushesBufferedEventsAfterCommit(t *testing.T) {
	hub := &fakeBroadcaster{}
	resolver := newFakeResolver()
	resolver.users[7] = []string{"u1", "u2"}
	emitter := NewTxEmitter(&fakeTM{}, hub).WithParticipantResolver(resolver)

	err := emitter.WithinTransaction(context.Background(), func(ctx context.Context, _ database.Tx) error {
		emitter.Emit(ctx, Event{Type: EventChainStatusChanged, ChainID: 7})
		emitter.Emit(ctx, Event{Type: EventChainVoteChanged, ChainID: 7})
		return nil
	})
	if err != nil {
		t.Fatalf("WithinTransaction() error = %v", err)
	}

	if got := hub.countRoom("chain:7"); got != 2 {
		t.Fatalf("chain room publishes = %d, want 2", got)
	}
	if got := hub.countRoom("user:u1"); got != 2 {
		t.Fatalf("user:u1 publishes = %d, want 2", got)
	}
	if resolver.calls[7] != 1 {
		t.Fatalf("resolver calls = %d, want 1 (dedup per chain per flush)", resolver.calls[7])
	}
}

func TestTxEmitterDropsEventsOnRollback(t *testing.T) {
	hub := &fakeBroadcaster{}
	emitter := NewTxEmitter(&fakeTM{commitFailure: true}, hub)

	_ = emitter.WithinTransaction(context.Background(), func(ctx context.Context, _ database.Tx) error {
		emitter.Emit(ctx, Event{Type: EventChainCreated, ChainID: 1})
		return nil
	})

	if len(hub.records) != 0 {
		t.Fatalf("rollback must discard buffered events, got %v", hub.eventTypes())
	}
}

func TestTxEmitterNestedTransactionFlushesOnceInOrder(t *testing.T) {
	hub := &fakeBroadcaster{}
	emitter := NewTxEmitter(&fakeTM{}, hub)

	err := emitter.WithinTransaction(context.Background(), func(ctx context.Context, _ database.Tx) error {
		emitter.Emit(ctx, Event{Type: EventChainCreated, ChainID: 1})
		return emitter.WithinTransaction(ctx, func(innerCtx context.Context, _ database.Tx) error {
			emitter.Emit(innerCtx, Event{Type: EventChainVoteChanged, ChainID: 1})
			return nil
		})
	})
	if err != nil {
		t.Fatalf("WithinTransaction() error = %v", err)
	}

	got := hub.eventTypes()
	want := []string{EventChainCreated, EventChainVoteChanged}
	if len(got) != len(want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("events = %v, want %v", got, want)
		}
	}
}

func TestTxEmitterImmediateEmitOutsideTransaction(t *testing.T) {
	hub := &fakeBroadcaster{}
	emitter := NewTxEmitter(&fakeTM{}, hub)

	emitter.Emit(context.Background(), Event{Type: EventItemStatusChanged, UserID: "u9"})

	if got := hub.countRoom("user:u9"); got != 1 {
		t.Fatalf("user:u9 publishes = %d, want 1", got)
	}
}

// A chain being deleted is unresolvable after commit, so its participants are passed
// as explicit recipients.
func TestTxEmitterPublishesExplicitRecipientsOnce(t *testing.T) {
	hub := &fakeBroadcaster{}
	emitter := NewTxEmitter(&fakeTM{}, hub) // no resolver: the rows are gone after the commit

	err := emitter.WithinTransaction(context.Background(), func(ctx context.Context, _ database.Tx) error {
		emitter.Emit(ctx, Event{
			Type:    EventChainRemoved,
			ChainID: 3,
			UserIDs: []string{"u1", "u2", "u1"},
		})
		return nil
	})
	if err != nil {
		t.Fatalf("WithinTransaction() error = %v", err)
	}

	if got := hub.countRoom("user:u1"); got != 1 {
		t.Fatalf("user:u1 publishes = %d, want 1 (deduplicated)", got)
	}
	if got := hub.countRoom("user:u2"); got != 1 {
		t.Fatalf("user:u2 publishes = %d, want 1", got)
	}
	if got := hub.countRoom("chain:3"); got != 1 {
		t.Fatalf("chain:3 publishes = %d, want 1", got)
	}
}

// A failed nested scope must not leak its events into the outer commit.
func TestTxEmitterDropsEventsOfFailedNestedTransaction(t *testing.T) {
	hub := &fakeBroadcaster{}
	emitter := NewTxEmitter(&fakeTM{}, hub)

	err := emitter.WithinTransaction(context.Background(), func(ctx context.Context, _ database.Tx) error {
		emitter.Emit(ctx, Event{Type: EventChainCreated, ChainID: 1})
		// the caller swallows the nested failure and the outer transaction commits
		_ = emitter.WithinTransaction(ctx, func(innerCtx context.Context, _ database.Tx) error {
			emitter.Emit(innerCtx, Event{Type: EventChainVoteChanged, ChainID: 1})
			return errors.New("nested failure")
		})
		return nil
	})
	if err != nil {
		t.Fatalf("WithinTransaction() error = %v", err)
	}

	got := hub.eventTypes()
	if len(got) != 1 || got[0] != EventChainCreated {
		t.Fatalf("events = %v, want only %s", got, EventChainCreated)
	}
}
