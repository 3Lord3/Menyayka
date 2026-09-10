package db_test

import (
	"context"
	"strings"
	"testing"

	database "github.com/Avito-Team-Not-Found/menyayka/internal/core/database"
)

// TestConnect_UnreachableDB checks that the app returns a clear error when the DB
// is unreachable.
func TestConnect_UnreachableDB(t *testing.T) {
	ctx := context.Background()
	// Port 1 on localhost is guaranteed closed — the DB is unreachable.
	_, err := database.Connect(ctx, "postgres://user:pass@127.0.0.1:1/nonexistent?sslmode=disable")

	if err == nil {
		t.Fatal("expected an error when DB is unreachable, got nil")
	}
	if !strings.Contains(err.Error(), "unreachable") {
		t.Fatalf("expected a clear 'unreachable' message, got: %v", err)
	}
}
