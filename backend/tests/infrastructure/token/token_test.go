package token_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/Avito-Team-Not-Found/menyayka/pkg/token"
)

func TestGenerateTicketParseRoundTrip(t *testing.T) {
	svc := token.NewService("secret", time.Hour)
	userID := uuid.New()

	ticket, jti, err := svc.GenerateTicket(userID, "ws", time.Minute)
	if err != nil {
		t.Fatalf("GenerateTicket() error = %v", err)
	}
	if jti == "" {
		t.Fatal("jti must not be empty")
	}

	gotUser, gotJTI, err := svc.ParseTicket(ticket, "ws")
	if err != nil {
		t.Fatalf("ParseTicket() error = %v", err)
	}
	if gotUser != userID || gotJTI != jti {
		t.Fatalf("ParseTicket() = (%v, %q), want (%v, %q)", gotUser, gotJTI, userID, jti)
	}
}

func TestParseTicketRejectsWrongPurpose(t *testing.T) {
	svc := token.NewService("secret", time.Hour)
	ticket, _, _ := svc.GenerateTicket(uuid.New(), "ws", time.Minute)

	if _, _, err := svc.ParseTicket(ticket, "other"); err == nil {
		t.Fatal("ticket with a mismatching purpose must be rejected")
	}
}

func TestParseTicketRejectsSessionToken(t *testing.T) {
	svc := token.NewService("secret", time.Hour)
	session, err := svc.Generate(uuid.New())
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if _, _, err := svc.ParseTicket(session, "ws"); err == nil {
		t.Fatal("a session token without purpose must be rejected as a ws ticket")
	}
}

func TestParseTicketRejectsExpired(t *testing.T) {
	svc := token.NewService("secret", time.Hour)
	ticket, _, _ := svc.GenerateTicket(uuid.New(), "ws", -time.Minute)

	if _, _, err := svc.ParseTicket(ticket, "ws"); err == nil {
		t.Fatal("expired ticket must be rejected")
	}
}

func TestParseTicketRequiresExp(t *testing.T) {
	svc := token.NewService("secret", time.Hour)
	claims := jwt.MapClaims{
		"sub":     uuid.New().String(),
		"purpose": "ws",
		"jti":     uuid.NewString(),
	}
	unsigned, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("signing failed: %v", err)
	}

	if _, _, err := svc.ParseTicket(unsigned, "ws"); err == nil {
		t.Fatal("ticket without exp must be rejected")
	}
}
