// Package api defines the unified HTTP response format:
//
//	{"error": "...", "code": 400}
package api

import "github.com/gin-gonic/gin"

// ErrorResponse is the unified error body returned by the API.
type ErrorResponse struct {
	Error string `json:"error" example:"внутренняя ошибка сервера"`
	Code  int    `json:"code" example:"500"`
}

// SendError writes an error response and aborts the handler chain.
func SendError(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, ErrorResponse{Error: message, Code: status})
}

// SendOk writes a success response with the given HTTP status.
func SendOk(c *gin.Context, status int, data any) {
	c.JSON(status, data)
}
