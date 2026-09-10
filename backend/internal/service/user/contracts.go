package user

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Avito-Team-Not-Found/menyayka/internal/entity"
)

// Repository is what service/user expects from the storage layer.
// Implemented in internal/repository/user.
type Repository interface {
	Create(ctx context.Context, user *entity.User) error
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
}

// TokenIssuer is what service/user expects from session-token issuance.
// Implemented in internal/infrastructure/token.
type TokenIssuer interface {
	Generate(userID uuid.UUID) (string, error)
}

// CodeStore is what service/user expects from the password-recovery code store.
// Implemented in internal/infrastructure/codestore.
type CodeStore interface {
	Save(key, value string, ttl time.Duration)
	Get(key string) (string, bool)
	Delete(key string)
}

// Mailer is what service/user expects from the email infrastructure.
// Implemented in internal/infrastructure/mailer.
type Mailer interface {
	SendRecoveryCode(to, code string) error
}
