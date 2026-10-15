// Package mailer sends transactional emails (password recovery, exchange
// confirmations). Two backends are supported:
//
//   - smtp   — RFC 5321 SMTP. Use when your VPS / network allows outbound
//              port 25 / 465 / 587 and you have a relay (Mailgun, Brevo,
//              Gmail, your own SMTP server, ...).
//   - resend — Resend HTTP API (https://resend.com). Use when SMTP is
//              blocked (e.g. many cloud VPS providers) and you want a
//              hosted API instead.
//
// The choice is made at startup via MAILER_PROVIDER in the environment.
package mailer

import (
	"errors"
	"fmt"
)

// Provider selects which backend NewService builds.
type Provider string

const (
	ProviderSMTP   Provider = "smtp"
	ProviderResend Provider = "resend"
)

// Service is the contract used by the application code.
type Service interface {
	SendRecoveryCode(to, code string) error
	SendChainFrozen(to string, chainID int64, gives, receives string) error
	SendReplacementInvitation(to string, chainID int64, gives, receives string) error
}

// ErrNotConfigured is returned when the chosen provider has its required
// configuration missing (e.g. empty host for SMTP, empty API key for Resend).
var ErrNotConfigured = errors.New("mailer is not configured")

// SMTPConfig is used when Provider == "smtp".
type SMTPConfig struct {
	Host       string
	Port       string
	Username   string
	Password   string
	From       string
	Encryption string // plain | starttls | tls; empty = starttls
}

// ResendConfig is used when Provider == "resend".
type ResendConfig struct {
	APIKey string
	From   string
}

// Config bundles everything the factory needs.
type Config struct {
	Provider Provider
	SMTP     SMTPConfig
	Resend   ResendConfig
}

// NewService returns a Service backed by the configured provider.
//
// Returns an error only for an unknown provider — an empty / missing
// configuration is not fatal: the service starts and SendXxx return
// ErrNotConfigured at first use. This matches the historical behaviour
// (the backend used to start with no mailer configured).
func NewService(cfg Config) (Service, error) {
	switch cfg.Provider {
	case ProviderSMTP, "":
		return newSMTPService(smtpConfigFromBundle(cfg.SMTP)), nil
	case ProviderResend:
		return newResendService(cfg.Resend), nil
	default:
		return nil, fmt.Errorf("unsupported mailer provider %q", cfg.Provider)
	}
}