package chain

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Avito-Team-Not-Found/menyayka/internal/api"
	"github.com/Avito-Team-Not-Found/menyayka/internal/entity"
)

// List godoc
// @Summary      List my chains
// @Tags         chains
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   chainResponse
// @Failure      401  {object}  api.ErrorResponse
// @Failure      500  {object}  api.ErrorResponse
// @Router       /chains [get]
func (h *Handler) List(c *gin.Context) {
	userID, ok := api.CurrentUserString(c)
	if !ok {
		return
	}

	chains, err := h.service.List(c.Request.Context(), userID)
	if err != nil {
		api.SendError(c, http.StatusInternalServerError, "внутренняя ошибка сервера")
		return
	}

	response := make([]chainResponse, 0, len(chains))
	for _, chain := range chains {
		response = append(response, newChainResponse(chain, userID))
	}
	api.SendOk(c, http.StatusOK, response)
}

// ExchangeOptions godoc
// @Summary      List exchange options for an offer
// @Description  Returns ready fulfilment options for the owner's request.
// @Tags         exchange-offers
// @Produce      json
// @Security     BearerAuth
// @Param        id  path      int  true  "Exchange offer ID"
// @Success      200  {array}   exchangeOptionsResponse
// @Failure      401  {object}  api.ErrorResponse
// @Failure      404  {object}  api.ErrorResponse
// @Failure      500  {object}  api.ErrorResponse
// @Router       /exchange-offers/{id}/exchange-options [get]
func (h *Handler) ExchangeOptions(c *gin.Context) {
	userID, ok := api.CurrentUserString(c)
	if !ok {
		return
	}

	offerID, ok := api.PathInt64(c, "id")
	if !ok {
		return
	}

	chains, err := h.service.ListForOffer(c.Request.Context(), userID, offerID)
	if err != nil {
		if errors.Is(err, entity.ErrExchangeOfferNotFound) {
			api.SendError(c, http.StatusNotFound, err.Error())
			return
		}
		api.SendError(c, http.StatusInternalServerError, "внутренняя ошибка сервера")
		return
	}

	response := make([]exchangeOptionsResponse, 0, len(chains))
	for _, chain := range chains {
		response = append(response, newExchangeOptionsResponse(chain))
	}
	api.SendOk(c, http.StatusOK, response)
}

// Get godoc
// @Summary      Get chain
// @Tags         chains
// @Produce      json
// @Security     BearerAuth
// @Param        id  path      int  true  "Chain ID"
// @Success      200  {object}  chainResponse
// @Failure      401  {object}  api.ErrorResponse
// @Failure      404  {object}  api.ErrorResponse
// @Failure      410  {object}  api.ErrorResponse  "confirmation expired"
// @Failure      500  {object}  api.ErrorResponse
// @Router       /chains/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	userID, ok := api.CurrentUserString(c)
	if !ok {
		return
	}

	chainID, ok := api.PathInt64(c, "id")
	if !ok {
		return
	}

	chain, err := h.service.Get(c.Request.Context(), userID, chainID)
	if err != nil {
		if errors.Is(err, entity.ErrChainNotFound) {
			api.SendError(c, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, entity.ErrChainConfirmationExpired) {
			api.SendError(c, http.StatusGone, err.Error())
			return
		}
		api.SendError(c, http.StatusInternalServerError, "внутренняя ошибка сервера")
		return
	}
	api.SendOk(c, http.StatusOK, newChainResponse(chain, userID))
}

// ListReplacements godoc
// @Summary      List replacement options
// @Description  ACTIVE requests from the declining cluster that can replace a participant without rebuilding the cycle.
// @Tags         chains
// @Produce      json
// @Security     BearerAuth
// @Param        id  path      int  true  "Chain ID"
// @Success      200  {array}   replacementResponse
// @Failure      401  {object}  api.ErrorResponse
// @Failure      403  {object}  api.ErrorResponse
// @Failure      404  {object}  api.ErrorResponse
// @Failure      409  {object}  api.ErrorResponse
// @Failure      500  {object}  api.ErrorResponse
// @Router       /chains/{id}/replacements [get]
func (h *Handler) ListReplacements(c *gin.Context) {
	userID, ok := api.CurrentUserString(c)
	if !ok {
		return
	}
	chainID, ok := api.PathInt64(c, "id")
	if !ok {
		return
	}
	options, err := h.service.ListReplacements(c.Request.Context(), userID, chainID)
	if err != nil {
		DetermineError(c, err)
		return
	}
	response := make([]replacementResponse, 0, len(options))
	for _, option := range options {
		response = append(response, replacementResponse{
			RequestID: option.RequestID, OfferedItemID: option.OfferedItemID,
			Title: option.Title, Description: option.Description,
			WantedDescription: option.WantedDescription, ImageURL: option.ImageURL,
			Reliability: option.Reliability, RespondedAt: option.RespondedAt.UTC().Format(time.RFC3339),
		})
	}
	api.SendOk(c, http.StatusOK, response)
}
