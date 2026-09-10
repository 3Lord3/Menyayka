package embedding

import "context"

// Client converts text into a pgvector vector used by matching. The implementation
// may call an external service, so it runs before opening a DB transaction.
type Client interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}
