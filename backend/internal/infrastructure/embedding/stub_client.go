package embedding

import (
	"context"
	"crypto/sha256"
)

const stubVectorDimension = 384

// StubClient produces a deterministic vector of the required size without an external API.
// It's for local CRUD checks before wiring a real embeddings service; it doesn't do
// semantic text comparison.
type StubClient struct{}

// NewStubClient creates a local embeddings stub client.
func NewStubClient() *StubClient {
	return &StubClient{}
}

// Embed returns a deterministic 384-dim vector for the given text.
func (c *StubClient) Embed(_ context.Context, text string) ([]float32, error) {
	sum := sha256.Sum256([]byte(text))
	vector := make([]float32, stubVectorDimension)
	for i := range vector {
		vector[i] = float32(sum[i%len(sum)]^byte(i*31))/127.5 - 1
	}
	return vector, nil
}

var _ Client = (*StubClient)(nil)
