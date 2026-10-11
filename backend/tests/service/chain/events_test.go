package chain_test

import (
	"context"
	"testing"

	"github.com/Avito-Team-Not-Found/menyayka/internal/entity"
	"github.com/Avito-Team-Not-Found/menyayka/internal/realtime"
	chainservice "github.com/Avito-Team-Not-Found/menyayka/internal/service/chain"
)

type recordingEmitter struct {
	events []realtime.Event
}

func (e *recordingEmitter) Emit(_ context.Context, event realtime.Event) {
	e.events = append(e.events, event)
}

func (e *recordingEmitter) count(eventType string) int {
	n := 0
	for _, event := range e.events {
		if event.Type == eventType {
			n++
		}
	}
	return n
}

// Whoever actually changed the chain emits chain.expired exactly once; a concurrent
// path that sees expired=false stays silent.
func TestExpireDueEmitsExpiredOncePerChangedChain(t *testing.T) {
	repository := &fakeRepository{
		expiredChainIDs:       []int64{7},
		expiredFrozen:         true,
		expiredFrozenRequests: []int64{10, 20},
	}
	freezer := chainservice.NewFreezeService(repository, &fakeRebuilder{})
	emitter := &recordingEmitter{}
	service := chainservice.NewService(repository, fakeTransactionManager{}).
		WithFreezer(freezer).
		WithRealtimeEmitter(emitter)

	if err := service.ExpireDue(context.Background()); err != nil {
		t.Fatalf("ExpireDue() error = %v", err)
	}

	if got := emitter.count(realtime.EventChainExpired); got != 1 {
		t.Fatalf("chain.expired events = %d, want 1", got)
	}
	if got := emitter.count(realtime.EventChainStatusChanged); got != 1 {
		t.Fatalf("chain.status_changed events = %d, want 1", got)
	}
	for _, event := range emitter.events {
		if event.ChainID != 7 {
			t.Fatalf("event %s chainId = %d, want 7", event.Type, event.ChainID)
		}
	}
}

// Nothing overdue means no event, so a no-op poll never invalidates caches.
func TestExpireDueStaysSilentWhenNothingExpired(t *testing.T) {
	repository := &fakeRepository{}
	freezer := chainservice.NewFreezeService(repository, &fakeRebuilder{})
	emitter := &recordingEmitter{}
	service := chainservice.NewService(repository, fakeTransactionManager{}).
		WithFreezer(freezer).
		WithRealtimeEmitter(emitter)

	if err := service.ExpireDue(context.Background()); err != nil {
		t.Fatalf("ExpireDue() error = %v", err)
	}
	if len(emitter.events) != 0 {
		t.Fatalf("events = %v, want none", emitter.events)
	}
}

// chain.created is emitted once per inserted chain; deduplicated drafts produce none.
func TestSaveCandidatesEmitsChainCreatedOncePerInsertedChain(t *testing.T) {
	repository := &fakeRepository{createdChains: []int64{11, 12}}
	emitter := &recordingEmitter{}
	service := chainservice.NewService(repository, fakeTransactionManager{}).
		WithRealtimeEmitter(emitter)
	draft := entity.ChainDraft{
		Score: 0.9,
		Participants: []entity.ChainDraftParticipant{
			{ClusterID: 10, RequestID: 1},
			{ClusterID: 20, RequestID: 2},
		},
	}

	if err := service.SaveCandidates(context.Background(), nil, []entity.ChainDraft{draft}); err != nil {
		t.Fatalf("SaveCandidates() error = %v", err)
	}

	if got := emitter.count(realtime.EventChainCreated); got != 2 {
		t.Fatalf("chain.created events = %d, want 2", got)
	}
	seen := map[int64]bool{}
	for _, event := range emitter.events {
		if seen[event.ChainID] {
			t.Fatalf("chain %d emitted twice", event.ChainID)
		}
		seen[event.ChainID] = true
	}
}

func TestSaveCandidatesSkipsEmissionWhenNothingInserted(t *testing.T) {
	repository := &fakeRepository{} // dedup path: no created IDs
	emitter := &recordingEmitter{}
	service := chainservice.NewService(repository, fakeTransactionManager{}).
		WithRealtimeEmitter(emitter)
	draft := entity.ChainDraft{
		Score: 0.9,
		Participants: []entity.ChainDraftParticipant{
			{ClusterID: 10, RequestID: 1},
			{ClusterID: 20, RequestID: 2},
		},
	}

	if err := service.SaveCandidates(context.Background(), nil, []entity.ChainDraft{draft}); err != nil {
		t.Fatalf("SaveCandidates() error = %v", err)
	}
	if got := emitter.count(realtime.EventChainCreated); got != 0 {
		t.Fatalf("chain.created events = %d, want 0", got)
	}
}
