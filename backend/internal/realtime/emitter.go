package realtime

import "context"

// Emitter publishes realtime events; services depend on this, not on the Hub.
type Emitter interface {
	Emit(ctx context.Context, event Event)
}

// Broadcaster delivers a message to every client in the given rooms.
type Broadcaster interface {
	PublishRooms(rooms []string, msg []byte)
}

// ParticipantResolver maps a chain to its participants' user IDs; called only at
// flush time, after commit.
type ParticipantResolver interface {
	ListChainParticipantUserIDs(ctx context.Context, chainID int64) ([]string, error)
}

// ChainAccessChecker reports whether a user takes part in a chain; subscriptions are
// gated on it.
type ChainAccessChecker interface {
	IsChainParticipant(ctx context.Context, userID string, chainID int64) (bool, error)
}
