package mailer_test

import (
	"errors"
	"testing"

	"github.com/Avito-Team-Not-Found/menyayka/internal/infrastructure/mailer"
)

func TestNewService_UnknownProvider(t *testing.T) {
	_, err := mailer.NewService(mailer.Config{Provider: "sendgrid"})
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestNewService_EmptyConfigStarts(t *testing.T) {
	// Both backends should start without error even if nothing is configured;
	// they only fail at SendXxx time. This matches the historical behaviour:
	// the backend boots without mailer and only password-recovery fails.
	for _, p := range []mailer.Provider{mailer.ProviderSMTP, mailer.ProviderResend, ""} {
		svc, err := mailer.NewService(mailer.Config{Provider: p})
		if err != nil {
			t.Fatalf("Provider=%q: NewService err = %v", p, err)
		}
		if svc == nil {
			t.Fatalf("Provider=%q: nil service", p)
		}
	}
}

func TestSend_NotConfiguredAtRuntime(t *testing.T) {
	svc, err := mailer.NewService(mailer.Config{Provider: mailer.ProviderSMTP})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.SendRecoveryCode("ivan@example.com", "123456"); !errors.Is(err, mailer.ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured, got %v", err)
	}
	if err := svc.SendChainFrozen("ivan@example.com", 7, "PlayStation 5", "iPhone 14"); !errors.Is(err, mailer.ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured, got %v", err)
	}
	if err := svc.SendReplacementInvitation("ivan@example.com", 7, "PlayStation 5", "iPhone 14"); !errors.Is(err, mailer.ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured, got %v", err)
	}

	svcResend, err := mailer.NewService(mailer.Config{
		Provider: mailer.ProviderResend,
		Resend:   mailer.ResendConfig{APIKey: "", From: "noreply@example.com"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svcResend.SendRecoveryCode("ivan@example.com", "123456"); !errors.Is(err, mailer.ErrNotConfigured) {
		t.Fatalf("Resend without API key should ErrNotConfigured at send, got %v", err)
	}
}

func TestSend_InvalidAddress(t *testing.T) {
	svc, err := mailer.NewService(mailer.Config{
		Provider: mailer.ProviderSMTP,
		SMTP: mailer.SMTPConfig{
			Host:     "smtp.example.com",
			Port:     "587",
			Username: "u",
			Password: "p",
			From:     "noreply@example.com",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range []struct {
		name string
		call func() error
	}{
		{"recovery", func() error { return svc.SendRecoveryCode("not-an-email", "123456") }},
		{"frozen", func() error { return svc.SendChainFrozen("not-an-email", 7, "PlayStation 5", "iPhone 14") }},
		{"replacement", func() error { return svc.SendReplacementInvitation("not-an-email", 7, "PlayStation 5", "iPhone 14") }},
	} {
		if err := c.call(); err == nil {
			t.Fatalf("%s: expected error for invalid recipient address", c.name)
		}
	}
}

func TestResendService_EmptyFrom(t *testing.T) {
	svc, err := mailer.NewService(mailer.Config{
		Provider: mailer.ProviderResend,
		Resend:   mailer.ResendConfig{APIKey: "re_test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.SendRecoveryCode("ivan@example.com", "123456"); err == nil {
		t.Fatal("expected error when From is empty")
	}
}
