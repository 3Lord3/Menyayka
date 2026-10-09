package realtime

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Avito-Team-Not-Found/menyayka/internal/core/database"
)

type ctxBufferKey struct{}

// txBuffer accumulates events emitted inside a transaction and flushes to the Hub
// only after commit. Nested transactions get their own segment, so a failed nested
// call drops exactly its own events even if the outer transaction commits.
type txBuffer struct {
	mu     sync.Mutex
	events []Event
}

// absorb appends a finished child segment to this buffer, preserving emission order.
func (b *txBuffer) absorb(child *txBuffer) {
	child.mu.Lock()
	events := child.events
	child.events = nil
	child.mu.Unlock()
	if len(events) == 0 {
		return
	}
	b.mu.Lock()
	b.events = append(b.events, events...)
	b.mu.Unlock()
}

func bufferFrom(ctx context.Context) (*txBuffer, bool) {
	buf, ok := ctx.Value(ctxBufferKey{}).(*txBuffer)
	return buf, ok
}

func withBuffer(ctx context.Context, buf *txBuffer) context.Context {
	return context.WithValue(ctx, ctxBufferKey{}, buf)
}

// TxEmitter is both a TransactionManager and an Emitter: events in a tx-scoped
// context are buffered and flushed after commit, others go straight to the Hub.
type TxEmitter struct {
	inner    database.TransactionManager
	hub      Broadcaster
	resolver ParticipantResolver
	logger   *logrus.Logger
}

// NewTxEmitter wraps a transaction manager so services depend on one object for
// transactions and realtime events.
func NewTxEmitter(inner database.TransactionManager, hub Broadcaster) *TxEmitter {
	return &TxEmitter{inner: inner, hub: hub}
}

// WithParticipantResolver sets the chain-to-participants lookup used at flush time.
func (e *TxEmitter) WithParticipantResolver(resolver ParticipantResolver) *TxEmitter {
	e.resolver = resolver
	return e
}

// WithLogger sets an optional logger for flush-time resolution errors.
func (e *TxEmitter) WithLogger(logger *logrus.Logger) *TxEmitter {
	e.logger = logger
	return e
}

// WithinTransaction runs fn in a transaction with a buffer that Emit fills. Only the
// outermost call flushes; a failed nested call drops its own segment.
func (e *TxEmitter) WithinTransaction(ctx context.Context, fn func(context.Context, database.Tx) error) error {
	parent, nested := bufferFrom(ctx)
	buf := &txBuffer{}
	err := e.inner.WithinTransaction(ctx, func(txCtx context.Context, tx database.Tx) error {
		return fn(withBuffer(txCtx, buf), tx)
	})
	if err != nil {
		return err
	}
	if nested {
		parent.absorb(buf)
		return nil
	}
	e.flush(ctx, buf)
	return nil
}

// Emit buffers the event when called inside a transaction and dispatches it
// immediately otherwise.
func (e *TxEmitter) Emit(ctx context.Context, event Event) {
	if event.At.IsZero() {
		event.At = time.Now()
	}
	if buf, ok := bufferFrom(ctx); ok {
		buf.mu.Lock()
		buf.events = append(buf.events, event)
		buf.mu.Unlock()
		return
	}
	e.flush(ctx, &txBuffer{events: []Event{event}})
}

func (e *TxEmitter) flush(ctx context.Context, buf *txBuffer) {
	buf.mu.Lock()
	events := buf.events
	buf.events = nil
	buf.mu.Unlock()
	if len(events) == 0 {
		return
	}

	// Post-commit delivery must not depend on the caller's lifetime.
	resolveCtx := context.WithoutCancel(ctx)
	resolved := make(map[int64][]string)
	for _, event := range events {
		data, err := marshal(event)
		if err != nil {
			if e.logger != nil {
				e.logger.WithError(err).WithField("event_type", event.Type).Error("failed to marshal realtime event")
			}
			continue
		}

		rooms := make([]string, 0, 2+len(event.UserIDs))
		recipients := make([]string, 0, len(event.UserIDs))
		recipients = append(recipients, event.UserIDs...)
		if event.ChainID > 0 {
			rooms = append(rooms, chainRoom(event.ChainID))
			participants, ok := resolved[event.ChainID]
			if !ok {
				participants = e.resolveParticipants(resolveCtx, event.ChainID)
				resolved[event.ChainID] = participants
			}
			recipients = append(recipients, participants...)
		}
		if event.UserID != "" {
			recipients = append(recipients, event.UserID)
		}
		for _, userID := range dedup(recipients) {
			rooms = append(rooms, userRoom(userID))
		}
		// One publish per event: a client in several matched rooms gets it once.
		e.hub.PublishRooms(rooms, data)
	}
}

// dedup removes duplicates while keeping the first-seen order.
func dedup(values []string) []string {
	if len(values) < 2 {
		return values
	}
	seen := make(map[string]struct{}, len(values))
	out := values[:0]
	for _, v := range values {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func (e *TxEmitter) resolveParticipants(ctx context.Context, chainID int64) []string {
	if e.resolver == nil {
		return nil
	}
	userIDs, err := e.resolver.ListChainParticipantUserIDs(ctx, chainID)
	if err != nil {
		if e.logger != nil {
			e.logger.WithError(err).WithField("chain_id", chainID).Error("failed to resolve chain participants for realtime event")
		}
		return nil
	}
	return userIDs
}

var (
	_ database.TransactionManager = (*TxEmitter)(nil)
	_ Emitter                     = (*TxEmitter)(nil)
)
