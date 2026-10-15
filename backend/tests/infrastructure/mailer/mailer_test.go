package mailer_test

import (
	"errors"
	"testing"

	"github.com/Avito-Team-Not-Found/menyayka/internal/infrastructure/mailer"
)

func newUnconfigured(t *testing.T) *mailer.Service {
	t.Helper()
	return mailer.NewService(mailer.Config{})
}

func TestSendRecoveryCode_NotConfigured(t *testing.T) {
	err := newUnconfigured(t).SendRecoveryCode("ivan@example.com", "123456")
	if !errors.Is(err, mailer.ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured, got %v", err)
	}
}

func TestSendRecoveryCode_InvalidAddress(t *testing.T) {
	svc := mailer.NewService(mailer.Config{
		APIKey: "re_test_key",
		From:   "noreply@example.com",
	})

	err := svc.SendRecoveryCode("not-an-email", "123456")
	if err == nil {
		t.Fatal("expected error for invalid recipient address")
	}
}

func TestSendRecoveryCode_EmptyFrom(t *testing.T) {
	svc := mailer.NewService(mailer.Config{
		APIKey: "re_test_key",
	})

	err := svc.SendRecoveryCode("ivan@example.com", "123456")
	if err == nil {
		t.Fatal("expected error when From is empty")
	}
}

func TestSendChainFrozen_NotConfigured(t *testing.T) {
	err := newUnconfigured(t).SendChainFrozen("ivan@example.com", 7, "PlayStation 5", "iPhone 14")
	if !errors.Is(err, mailer.ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured, got %v", err)
	}
}

func TestSendReplacementInvitation_NotConfigured(t *testing.T) {
	err := newUnconfigured(t).SendReplacementInvitation("ivan@example.com", 7, "PlayStation 5", "iPhone 14")
	if !errors.Is(err, mailer.ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured, got %v", err)
	}
}