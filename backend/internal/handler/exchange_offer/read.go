package exchange_offer

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Avito-Team-Not-Found/menyayka/internal/api"
	"github.com/Avito-Team-Not-Found/menyayka/internal/entity"
)

// Get godoc
// @Summary      Get exchange offer
// @Tags         exchange-offers
// @Produce      json
// @Security     BearerAuth
// @Param        id  path      int  true  "Offer ID"
// @Success      200  {object}  exchangeOfferResponse
// @Failure      401  {object}  api.ErrorResponse
// @Failure      403  {object}  api.ErrorResponse  "not the owner"
// @Failure      404  {object}  api.ErrorResponse
// @Failure      500  {object}  api.ErrorResponse
// @Router       /exchange-offers/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	userID, ok := api.CurrentUserString(c)
	if !ok {
		return
	}

	requestID, ok := api.PathInt64(c, "id")
	if !ok {
		return
	}

	request, err := h.service.Get(c.Request.Context(), userID, requestID)
	if err != nil {
		switch {
		case errors.Is(err, entity.ErrExchangeOfferNotFound):
			api.SendError(c, http.StatusNotFound, err.Error())
		case errors.Is(err, entity.ErrExchangeOfferForbidden):
			api.SendError(c, http.StatusForbidden, err.Error())
		default:
			api.SendError(c, http.StatusInternalServerError, "внутренняя ошибка сервера")
		}
		return
	}

	api.SendOk(c, http.StatusOK, newExchangeOfferResponse(request))
}

// List godoc
// @Summary      List my exchange offers
// @Tags         exchange-offers
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   exchangeOfferListResponse
// @Failure      401  {object}  api.ErrorResponse
// @Failure      500  {object}  api.ErrorResponse
// @Router       /exchange-offers [get]
func (h *Handler) List(c *gin.Context) {
	userID, ok := api.CurrentUserString(c)
	if !ok {
		return
	}

	requests, err := h.service.List(c.Request.Context(), userID)
	if err != nil {
		api.SendError(c, http.StatusInternalServerError, "внутренняя ошибка сервера")
		return
	}

	response := make([]exchangeOfferListResponse, 0, len(requests))
	for _, request := range requests {
		response = append(response, exchangeOfferListResponse{
			exchangeOfferResponse: newExchangeOfferResponse(request.ExchangeOffer),
			OfferedItemTitle:      request.OfferedItemTitle,
		})
	}

	api.SendOk(c, http.StatusOK, response)
}
