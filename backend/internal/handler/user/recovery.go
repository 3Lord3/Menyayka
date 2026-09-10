package user

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Avito-Team-Not-Found/menyayka/internal/api"
	"github.com/Avito-Team-Not-Found/menyayka/internal/entity"
)

type sendRecoveryCodeRequest struct {
	Email string `json:"email" validate:"required,email,max=255"`
}

// SendRecoveryCode godoc
// @Summary      Send password recovery code
// @Description  Emails a one-time recovery code (MVP: in-memory store, no rate limiting).
// @Tags         password-recovery
// @Accept       json
// @Produce      json
// @Param        body  body      sendRecoveryCodeRequest  true  "Recovery email"
// @Success      200   {object}  map[string]string  "{\"message\": \"code_sent\"}"
// @Failure      404   {object}  api.ErrorResponse  "user not found"
// @Failure      422   {object}  api.ErrorResponse  "validation error"
// @Failure      500   {object}  api.ErrorResponse  "SMTP not configured / send failed"
// @Router       /account/password-recovery/send-code/ [post]
func (h *Handler) SendRecoveryCode(c *gin.Context) {
	var req sendRecoveryCodeRequest
	if !api.BindJSON(c, &req) {
		return
	}

	if err := h.service.SendRecoveryCode(c.Request.Context(), req.Email); err != nil {
		switch {
		case errors.Is(err, entity.ErrUserNotFound):
			api.SendError(c, http.StatusNotFound, "пользователь с таким email не найден")
		default:
			api.SendError(c, http.StatusInternalServerError, "не удалось отправить код на почту")
		}
		return
	}

	api.SendOk(c, http.StatusOK, gin.H{"message": "code_sent"})
}

type verifyRecoveryCodeRequest struct {
	Email string `json:"email" validate:"required,email,max=255"`
	Code  string `json:"code" validate:"required,recovery_code"`
}

// VerifyRecoveryCode godoc
// @Summary      Verify recovery code
// @Tags         password-recovery
// @Accept       json
// @Produce      json
// @Param        body  body      verifyRecoveryCodeRequest  true  "Email + code"
// @Success      200   {object}  map[string]string  "{\"message\": \"code_valid\"}"
// @Failure      400   {object}  api.ErrorResponse  "invalid or expired code"
// @Failure      422   {object}  api.ErrorResponse  "validation error"
// @Failure      500   {object}  api.ErrorResponse
// @Router       /account/password-recovery/verify-code/ [post]
func (h *Handler) VerifyRecoveryCode(c *gin.Context) {
	var req verifyRecoveryCodeRequest
	if !api.BindJSON(c, &req) {
		return
	}

	if err := h.service.VerifyRecoveryCode(c.Request.Context(), req.Email, req.Code); err != nil {
		switch {
		case errors.Is(err, entity.ErrInvalidRecoveryCode):
			api.SendError(c, http.StatusBadRequest, "неверный или истёкший код")
		default:
			api.SendError(c, http.StatusInternalServerError, "внутренняя ошибка сервера")
		}
		return
	}

	api.SendOk(c, http.StatusOK, gin.H{"message": "code_valid"})
}

type resetPasswordRequest struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Code     string `json:"code" validate:"required,recovery_code"`
	Password string `json:"password" validate:"required,min=8"`
}

// Re-checks the code authoritatively, not relying on a prior VerifyRecoveryCode call.
// ResetPassword godoc
// @Summary      Reset password
// @Description  Re-checks the code authoritatively, not relying on a prior verify call.
// @Tags         password-recovery
// @Accept       json
// @Produce      json
// @Param        body  body      resetPasswordRequest  true  "Email + code + new password"
// @Success      200   {object}  map[string]string  "{\"message\": \"password_changed\"}"
// @Failure      400   {object}  api.ErrorResponse  "invalid or expired code"
// @Failure      404   {object}  api.ErrorResponse  "user not found"
// @Failure      422   {object}  api.ErrorResponse  "validation error"
// @Failure      500   {object}  api.ErrorResponse
// @Router       /account/password-recovery/reset-password/ [post]
func (h *Handler) ResetPassword(c *gin.Context) {
	var req resetPasswordRequest
	if !api.BindJSON(c, &req) {
		return
	}

	err := h.service.ResetPassword(c.Request.Context(), req.Email, req.Code, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, entity.ErrInvalidRecoveryCode):
			api.SendError(c, http.StatusBadRequest, "неверный или истёкший код")
		case errors.Is(err, entity.ErrUserNotFound):
			api.SendError(c, http.StatusNotFound, "пользователь с таким email не найден")
		default:
			api.SendError(c, http.StatusInternalServerError, "внутренняя ошибка сервера")
		}
		return
	}

	api.SendOk(c, http.StatusOK, gin.H{"message": "password_changed"})
}
