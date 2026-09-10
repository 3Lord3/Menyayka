package item_test

import (
	"testing"

	itemrepo "github.com/Avito-Team-Not-Found/menyayka/internal/repository/item"
)

func TestNewRepository(t *testing.T) {
	if itemrepo.NewRepository(nil) == nil {
		t.Fatal("NewRepository(nil) returned nil")
	}
}
