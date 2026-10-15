// Resend HTTP API backend for the mailer package. Implements the Service
// interface using github.com/resend/resend-go.
//
// We use this when the deployment environment blocks outbound SMTP
// (25 / 465 / 587) — common on cloud VPS providers (Oracle Cloud, etc.).
package mailer

import (
	"context"
	"fmt"
	"net/mail"
	"time"

	"github.com/resend/resend-go/v4"
)

type resendService struct {
	cfg    ResendConfig
	client *resend.Client
}

func newResendService(cfg ResendConfig) *resendService {
	return &resendService{cfg: cfg, client: resend.NewClient(cfg.APIKey)}
}

func (r *resendService) SendRecoveryCode(to, code string) error {
	const subject = "Код для восстановления пароля"
	body := fmt.Sprintf(
		"Здравствуйте!\r\n\r\nВаш код для восстановления пароля: %s\r\nКод действителен 10 минут.\r\n\r\nЕсли вы не запрашивали восстановление пароля — проигнорируйте это письмо.",
		code,
	)
	return r.send(to, subject, body)
}

// SendChainFrozen informs a participant that every member confirmed the exchange.
func (r *resendService) SendChainFrozen(to string, chainID int64, gives, receives string) error {
	return r.send(
		to,
		"Обмен подтверждён всеми участниками",
		fmt.Sprintf("Цепочка обмена №%d собрана и заморожена.\r\n\r\nВы отдаёте: %s\r\nВы получаете: %s\r\n\r\nВсе участники подтвердили участие, товары зарезервированы для этой сделки.", chainID, gives, receives),
	)
}

// SendReplacementInvitation asks a candidate to explicitly accept a fast replacement.
func (r *resendService) SendReplacementInvitation(to string, chainID int64, gives, receives string) error {
	return r.send(
		to,
		"Приглашение в обмен",
		fmt.Sprintf("Вас пригласили заменить участника в цепочке обмена №%d.\r\n\r\nВы отдаёте: %s\r\nВы получаете: %s\r\n\r\nОткройте предложение в приложении и самостоятельно подтвердите участие или откажитесь.", chainID, gives, receives),
	)
}

func (r *resendService) send(to, subject, body string) error {
	if _, err := mail.ParseAddress(to); err != nil {
		return fmt.Errorf("invalid recipient address %q: %w", to, err)
	}

	if r.cfg.APIKey == "" || r.client == nil {
		return ErrNotConfigured
	}
	if r.cfg.From == "" {
		return fmt.Errorf("resend mailer: From is empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err := r.client.Emails.SendWithContext(ctx, &resend.SendEmailRequest{
		From:    r.cfg.From,
		To:      []string{to},
		Subject: subject,
		Text:    body,
	})
	if err != nil {
		return fmt.Errorf("resend send: %w", err)
	}
	return nil
}
