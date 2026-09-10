package user

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Avito-Team-Not-Found/menyayka/internal/api"
	"github.com/Avito-Team-Not-Found/menyayka/internal/entity"
	"github.com/Avito-Team-Not-Found/menyayka/internal/middleware"
)

type changePasswordRequest struct {
	CurrentPassword         string `json:"currentPassword" validate:"required"`
	NewPassword             string `json:"newPassword" validate:"required,min=8"`
	NewPasswordConfirmation string `json:"newPasswordConfirmation" validate:"required,eqfield=NewPassword"`
}

// ChangePassword godoc
// @Summary      Change password
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      changePasswordRequest  true  "Password change payload"
// @Success      200   {object}  map[string]string  "{\"message\": \"password_changed\"}"
// @Failure      400   {object}  api.ErrorResponse  "wrong current password"
// @Failure      401   {object}  api.ErrorResponse
// @Failure      404   {object}  api.ErrorResponse
// @Failure      422   {object}  api.ErrorResponse  "validation error"
// @Failure      500   {object}  api.ErrorResponse
// @Router       /auth/change-password [post]
func (h *Handler) ChangePassword(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		api.SendError(c, http.StatusUnauthorized, "требуется авторизация")
		return
	}

	var req changePasswordRequest
	if !api.BindJSON(c, &req) {
		return
	}

	err := h.service.ChangePassword(c.Request.Context(), userID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, entity.ErrInvalidCredentials):
			api.SendError(c, http.StatusBadRequest, "неверный текущий пароль")
		case errors.Is(err, entity.ErrUserNotFound):
			api.SendError(c, http.StatusNotFound, "пользователь не найден")
		default:
			api.SendError(c, http.StatusInternalServerError, "внутренняя ошибка сервера")
		}
		return
	}

	api.SendOk(c, http.StatusOK, gin.H{"message": "password_changed"})
}
