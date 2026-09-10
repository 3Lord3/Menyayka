package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Avito-Team-Not-Found/menyayka/pkg/validator"
)

// UserIDContextKey is the gin.Context key under which Auth stores the owner's uuid.UUID.
const UserIDContextKey = "userID"

// CurrentUserUUID reads userID from the context. Returns 401 if absent.
func CurrentUserUUID(c *gin.Context) (uuid.UUID, bool) {
	v, exists := c.Get(UserIDContextKey)
	if !exists {
		SendError(c, http.StatusUnauthorized, "требуется авторизация")
		return uuid.Nil, false
	}
	id, ok := v.(uuid.UUID)
	if !ok {
		SendError(c, http.StatusUnauthorized, "требуется авторизация")
		return uuid.Nil, false
	}
	return id, true
}

// CurrentUserString returns userID as a string for services that don't pass UUIDs yet.
func CurrentUserString(c *gin.Context) (string, bool) {
	id, ok := CurrentUserUUID(c)
	if !ok {
		return "", false
	}
	return id.String(), true
}

// PathInt64 reads a positive int64 path parameter. Returns 422 on error.
func PathInt64(c *gin.Context, name string) (int64, bool) {
	value, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || value <= 0 {
		SendError(c, http.StatusUnprocessableEntity, name+" должен быть положительным целым числом")
		return 0, false
	}
	return value, true
}

// BindJSON decodes and validates the JSON body. Returns false if a response was already sent.
func BindJSON(c *gin.Context, dst any) bool {
	return BindJSONWithInvalidMessage(c, dst, "некорректный JSON")
}

// BindJSONWithInvalidMessage is like BindJSON but uses a custom message for decode errors.
func BindJSONWithInvalidMessage(c *gin.Context, dst any, invalidJSONMessage string) bool {
	if err := validator.BindJSON(dst, c.Request); err != nil {
		var ve validator.Error
		if errors.As(err, &ve) {
			SendError(c, http.StatusUnprocessableEntity, err.Error())
			return false
		}
		var jsonSyntaxErr *json.SyntaxError
		if errors.As(err, &jsonSyntaxErr) {
			SendError(c, http.StatusBadRequest, invalidJSONMessage)
			return false
		}
		// Decoder may return io.ErrUnexpectedEOF / plain EOF for truncated JSON.
		SendError(c, http.StatusBadRequest, invalidJSONMessage)
		return false
	}
	return true
}
