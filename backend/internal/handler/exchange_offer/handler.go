package exchange_offer

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Avito-Team-Not-Found/menyayka/internal/api"
	"github.com/Avito-Team-Not-Found/menyayka/internal/middleware"
)

// Handler serves CRUD HTTP requests for exchange requests.
// The user ID comes only from the JWT middleware, never from the request body.
type Handler struct {
	service exchangeOfferService
}

func NewHandler(service exchangeOfferService) *Handler {
	return &Handler{service: service}
}

func currentUserID(c *gin.Context) (string, bool) {
	userID, ok := middleware.UserID(c)
	if !ok {
		api.SendError(c, http.StatusUnauthorized, "authentication is required")
		return "", false
	}
	return userID.String(), true
}

func pathID(c *gin.Context) (int64, bool) {
	requestID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || requestID <= 0 {
		api.SendError(c, http.StatusUnprocessableEntity, "id must be a positive integer")
		return 0, false
	}
	return requestID, true
}
