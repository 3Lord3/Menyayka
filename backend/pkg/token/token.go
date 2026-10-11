// Package token issues and parses JWT session tokens. It's a thin infrastructure
// wrapper around golang-jwt, with no business logic.
package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var ErrInvalidToken = errors.New("invalid or expired token")

// Service issues and validates HS256 JWTs.
type Service struct {
	secret []byte
	ttl    time.Duration
}

func NewService(secret string, ttl time.Duration) *Service {
	return &Service{secret: []byte(secret), ttl: ttl}
}

// Generate issues a session token for the user userID.
func (s *Service) Generate(userID uuid.UUID) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub": userID.String(),
		"iat": now.Unix(),
		"exp": now.Add(s.ttl).Unix(),
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
}

// GenerateTicket issues a short-lived token for a non-session purpose (e.g. "ws"),
// returning the token and its jti for one-time-store burning.
func (s *Service) GenerateTicket(userID uuid.UUID, purpose string, ttl time.Duration) (string, string, error) {
	now := time.Now()
	jti := uuid.NewString()
	claims := jwt.MapClaims{
		"sub":     userID.String(),
		"purpose": purpose,
		"jti":     jti,
		"iat":     now.Unix(),
		"exp":     now.Add(ttl).Unix(),
	}

	tokenStr, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
	if err != nil {
		return "", "", err
	}
	return tokenStr, jti, nil
}

// ParseTicket verifies a one-time token and returns its subject and jti; it must carry
// a matching purpose, a jti and exp.
//
// Single use is NOT enforced here: the caller must burn the jti in an atomic store
// shared across replicas (see internal/realtime.Handler).
func (s *Service) ParseTicket(tokenStr, wantPurpose string) (uuid.UUID, string, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.secret, nil
	}, jwt.WithExpirationRequired(), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !token.Valid {
		return uuid.Nil, "", ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, "", ErrInvalidToken
	}
	if purpose, _ := claims["purpose"].(string); purpose != wantPurpose {
		return uuid.Nil, "", ErrInvalidToken
	}
	jti, _ := claims["jti"].(string)
	if jti == "" {
		return uuid.Nil, "", ErrInvalidToken
	}
	sub, _ := claims["sub"].(string)
	userID, err := uuid.Parse(sub)
	if err != nil {
		return uuid.Nil, "", ErrInvalidToken
	}
	return userID, jti, nil
}

// Parse verifies the token signature and expiry and returns the user ID from the
// sub claim.
func (s *Service) Parse(tokenStr string) (uuid.UUID, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.secret, nil
	})
	if err != nil || !token.Valid {
		return uuid.Nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, ErrInvalidToken
	}

	sub, _ := claims["sub"].(string)
	userID, err := uuid.Parse(sub)
	if err != nil {
		return uuid.Nil, ErrInvalidToken
	}

	return userID, nil
}
