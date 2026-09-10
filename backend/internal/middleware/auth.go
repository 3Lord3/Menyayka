// Package middleware holds cross-cutting gin middlewares (auth, etc.) not tied to a
// specific feature.
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Avito-Team-Not-Found/menyayka/internal/api"
)

type TokenParser interface {
	Parse(tokenStr string) (uuid.UUID, error)
}

// Auth checks the "Authorization: Bearer <jwt>" header and stores the authorized
// user's ID in gin.Context. Returns 401 and aborts on a missing/invalid token.
func Auth(parser TokenParser) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			api.SendError(c, http.StatusUnauthorized, "требуется авторизация")
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, bearerPrefix)

		userID, err := parser.Parse(tokenStr)
		if err != nil {
			api.SendError(c, http.StatusUnauthorized, "недействительный или истёкший токен")
			return
		}

		c.Set(api.UserIDContextKey, userID)
		c.Next()
	}
}

// UserID reads the userID set by the Auth middleware. ok=false means Auth wasn't
// applied to this route — a routing bug.
func UserID(c *gin.Context) (uuid.UUID, bool) {
	v, exists := c.Get(api.UserIDContextKey)
	if !exists {
		return uuid.Nil, false
	}
	id, ok := v.(uuid.UUID)
	return id, ok
}
