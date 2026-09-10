package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// TEIClient implements embedding.Client via HTTP requests to TEI.
type TEIClient struct {
	baseURL    string
	httpClient *http.Client
	maxChars   int
}

// NewTEIClient creates a TEI client. baseURL is like "http://tei:80".
// maxChars is the maximum input-text length in runes (<=0 = no truncation).
func NewTEIClient(baseURL string, timeout time.Duration, maxChars int) *TEIClient {
	return &TEIClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: timeout},
		maxChars:   maxChars,
	}
}

// Embed POSTs text to /embed and returns the first response vector. The text is
// truncated to maxChars runes without breaking a UTF-8 sequence.
func (c *TEIClient) Embed(ctx context.Context, text string) ([]float32, error) {
	trimmed := truncateUTF8(text, c.maxChars)

	payload, err := json.Marshal(map[string]string{"inputs": trimmed})
	if err != nil {
		return nil, fmt.Errorf("tei: marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/embed", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("tei: request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tei: do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("tei: status %d: %s", resp.StatusCode, body)
	}

	var out [][]float32
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&out); err != nil {
		return nil, fmt.Errorf("tei: decode: %w", err)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("tei: empty embeddings")
	}
	return out[0], nil
}

// truncateUTF8 truncates a string to max runes without splitting multibyte chars.
// If max <= 0 or the length already fits, it returns the string unchanged.
func truncateUTF8(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	// advance runes until max is reached
	n := 0
	for i := range s {
		if n == max {
			return s[:i]
		}
		n++
	}
	return s
}
