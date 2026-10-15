// Package mailer sends transactional emails via the Resend HTTP API
// (https://resend.com). Using a hosted API instead of SMTP works around
// ISPs / VPS providers that block outbound port 25 / 465 / 587.
package mailer

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"time"

	"github.com/resend/resend-go/v4"
)

// ErrNotConfigured is returned when RESEND_API_KEY is unset.
var ErrNotConfigured = errors.New("mailer is not configured")

type Config struct {
	// APIKey is the Resend API key (re_...).
	APIKey string
	// From is the sender address. Must be a domain you've verified in
	// the Resend dashboard (e.g. "Menyayka <noreply@menyayka.lorot.dev>").
	From string
}

type Service struct {
	cfg    Config
	client *resend.Client
}

func NewService(cfg Config) *Service {
	var c *resend.Client
	if cfg.APIKey != "" {
		c = resend.NewClient(cfg.APIKey)
	}
	return &Service{cfg: cfg, client: c}
}

// SendRecoveryCode sends a one-time password-recovery code.
func (c *Service) SendRecoveryCode(to, code string) error {
	const subject = "Код для восстановления пароля"
	body := fmt.Sprintf(
		"Здравствуйте!\r\n\r\nВаш код для восстановления пароля: %s\r\nКод действителен 10 минут.\r\n\r\nЕсли вы не запрашивали восстановление пароля — проигнорируйте это письмо.",
		code,
	)
	return c.send(to, subject, body)
}

// SendChainFrozen informs a participant that every member confirmed the exchange.
func (c *Service) SendChainFrozen(to string, chainID int64, gives, receives string) error {
	return c.send(
		to,
		"Обмен подтверждён всеми участниками",
		fmt.Sprintf("Цепочка обмена №%d собрана и заморожена.\r\n\r\nВы отдаёте: %s\r\nВы получаете: %s\r\n\r\nВсе участники подтвердили участие, товары зарезервированы для этой сделки.", chainID, gives, receives),
	)
}

// SendReplacementInvitation asks a candidate to explicitly accept a fast replacement.
func (c *Service) SendReplacementInvitation(to string, chainID int64, gives, receives string) error {
	return c.send(
		to,
		"Приглашение в обмен",
		fmt.Sprintf("Вас пригласили заменить участника в цепочке обмена №%d.\r\n\r\nВы отдаёте: %s\r\nВы получаете: %s\r\n\r\nОткройте предложение в приложении и самостоятельно подтвердите участие или откажитесь.", chainID, gives, receives),
	)
}

func (c *Service) send(to, subject, body string) error {
	if _, err := mail.ParseAddress(to); err != nil {
		return fmt.Errorf("invalid recipient address %q: %w", to, err)
	}

	if c.cfg.APIKey == "" || c.client == nil {
		return ErrNotConfigured
	}
	if c.cfg.From == "" {
		return fmt.Errorf("mailer config: From is empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err := c.client.Emails.SendWithContext(ctx, &resend.SendEmailRequest{
		From:    c.cfg.From,
		To:      []string{to},
		Subject: subject,
		Text:    body,
	})
	if err != nil {
		return fmt.Errorf("resend send: %w", err)
	}
	return nil
}