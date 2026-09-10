package user

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Avito-Team-Not-Found/menyayka/internal/api"
	"github.com/Avito-Team-Not-Found/menyayka/internal/entity"
	"github.com/Avito-Team-Not-Found/menyayka/internal/middleware"
)

type registerRequest struct {
	FullName string `json:"fullName" validate:"not_empty,max=100"`
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=8"`
}

// Register godoc
// @Summary      Register a new user
// @Description  Creates a user and returns a JWT session token.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      registerRequest  true  "Registration payload"
// @Success      201   {object}  sessionResponse
// @Failure      409   {object}  api.ErrorResponse  "email already registered"
// @Failure      422   {object}  api.ErrorResponse  "validation error"
// @Failure      500   {object}  api.ErrorResponse
// @Router       /auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req registerRequest
	if !api.BindJSON(c, &req) {
		return
	}

	user, token, err := h.service.Register(c.Request.Context(), req.FullName, req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, entity.ErrUserAlreadyExists):
			api.SendError(c, http.StatusConflict, "пользователь с таким email уже зарегистрирован")
		default:
			api.SendError(c, http.StatusInternalServerError, "внутренняя ошибка сервера")
		}
		return
	}

	api.SendOk(c, http.StatusCreated, sessionResponse{
		Token: token,
		User: userResponse{
			ID:       user.ID.String(),
			FullName: user.FullName,
			Email:    user.Email,
		},
	})
}

type loginRequest struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required"`
}

// Login godoc
// @Summary      Log in
// @Description  Validates credentials and returns a JWT session token.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      loginRequest  true  "Login payload"
// @Success      200   {object}  sessionResponse
// @Failure      401   {object}  api.ErrorResponse  "invalid email or password"
// @Failure      422   {object}  api.ErrorResponse  "validation error"
// @Failure      500   {object}  api.ErrorResponse
// @Router       /auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if !api.BindJSON(c, &req) {
		return
	}

	user, token, err := h.service.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, entity.ErrInvalidCredentials):
			api.SendError(c, http.StatusUnauthorized, "неверный email или пароль")
		default:
			api.SendError(c, http.StatusInternalServerError, "внутренняя ошибка сервера")
		}
		return
	}

	api.SendOk(c, http.StatusOK, sessionResponse{
		Token: token,
		User: userResponse{
			ID:       user.ID.String(),
			FullName: user.FullName,
			Email:    user.Email,
		},
	})
}

// Logout is stateless JWT: the server only validates the token (no revocation stored).
// Logout godoc
// @Summary      Log out
// @Description  Stateless JWT: the server only validates the token, nothing is revoked.
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]string  "{\"message\": \"logged_out\"}"
// @Failure      401  {object}  api.ErrorResponse
// @Router       /auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	if _, ok := middleware.UserID(c); !ok {
		api.SendError(c, http.StatusUnauthorized, "требуется авторизация")
		return
	}

	api.SendOk(c, http.StatusOK, gin.H{"message": "logged_out"})
}

// Me godoc
// @Summary      Current user
// @Description  Returns the profile of the authenticated user.
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  userResponse
// @Failure      401  {object}  api.ErrorResponse
// @Failure      404  {object}  api.ErrorResponse
// @Failure      500  {object}  api.ErrorResponse
// @Router       /auth/me [get]
func (h *Handler) Me(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		api.SendError(c, http.StatusUnauthorized, "требуется авторизация")
		return
	}

	user, err := h.service.Me(c.Request.Context(), userID)
	if err != nil {
		switch {
		case errors.Is(err, entity.ErrUserNotFound):
			api.SendError(c, http.StatusNotFound, "пользователь не найден")
		default:
			api.SendError(c, http.StatusInternalServerError, "внутренняя ошибка сервера")
		}
		return
	}

	api.SendOk(c, http.StatusOK, userResponse{
		ID:       user.ID.String(),
		FullName: user.FullName,
		Email:    user.Email,
	})
}
