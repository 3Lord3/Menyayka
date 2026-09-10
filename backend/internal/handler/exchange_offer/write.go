package exchange_offer

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Avito-Team-Not-Found/menyayka/internal/api"
	"github.com/Avito-Team-Not-Found/menyayka/internal/entity"
	offerservice "github.com/Avito-Team-Not-Found/menyayka/internal/service/exchange_offer"
	"github.com/Avito-Team-Not-Found/menyayka/pkg/validator"
)

// Create godoc
// @Summary      Create exchange offer
// @Tags         exchange-offers
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      createBody  true  "Offer payload"
// @Success      201   {object}  exchangeOfferResponse
// @Failure      401   {object}  api.ErrorResponse
// @Failure      422   {object}  api.ErrorResponse  "validation error / offered item unavailable"
// @Failure      500   {object}  api.ErrorResponse
// @Router       /exchange-offers [post]
func (h *Handler) Create(c *gin.Context) {
	userID, ok := api.CurrentUserString(c)
	if !ok {
		return
	}

	var body createBody
	if !api.BindJSON(c, &body) {
		return
	}

	created, err := h.service.Create(c.Request.Context(), userID, offerservice.CreateInput{
		OfferedItemID:     body.OfferedItemID,
		WantedDescription: body.WantedDescription,
		WantedCategory:    body.WantedCategory,
	})
	if err != nil {
		var ve validator.Error
		switch {
		case errors.As(err, &ve):
			api.SendError(c, http.StatusUnprocessableEntity, err.Error())
		case errors.Is(err, entity.ErrOfferedItemUnavailable):
			api.SendError(c, http.StatusUnprocessableEntity, err.Error())
		default:
			api.SendError(c, http.StatusInternalServerError, "внутренняя ошибка сервера")
		}
		return
	}

	api.SendOk(c, http.StatusCreated, newExchangeOfferResponse(created))
}

// Update godoc
// @Summary      Update exchange offer
// @Tags         exchange-offers
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int         true  "Offer ID"
// @Param        body  body      updateBody  true  "Offer payload (optimistic locking via version)"
// @Success      200   {object}  exchangeOfferResponse
// @Failure      401   {object}  api.ErrorResponse
// @Failure      403   {object}  api.ErrorResponse  "not the owner"
// @Failure      404   {object}  api.ErrorResponse
// @Failure      409   {object}  api.ErrorResponse  "version conflict / offer locked"
// @Failure      422   {object}  api.ErrorResponse
// @Failure      500   {object}  api.ErrorResponse
// @Router       /exchange-offers/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	userID, ok := api.CurrentUserString(c)
	if !ok {
		return
	}

	requestID, ok := api.PathInt64(c, "id")
	if !ok {
		return
	}

	var body updateBody
	if !api.BindJSON(c, &body) {
		return
	}

	updated, err := h.service.Update(c.Request.Context(), userID, requestID, offerservice.UpdateInput{
		OfferedItemID:     body.OfferedItemID,
		WantedDescription: body.WantedDescription,
		WantedCategory:    body.WantedCategory,
		Version:           body.Version,
	})
	if err != nil {
		var ve validator.Error
		switch {
		case errors.As(err, &ve):
			api.SendError(c, http.StatusUnprocessableEntity, err.Error())
		case errors.Is(err, entity.ErrExchangeOfferNotFound):
			api.SendError(c, http.StatusNotFound, err.Error())
		case errors.Is(err, entity.ErrExchangeOfferForbidden):
			api.SendError(c, http.StatusForbidden, err.Error())
		case errors.Is(err, entity.ErrExchangeOfferVersionConflict),
			errors.Is(err, entity.ErrExchangeOfferLocked):
			api.SendError(c, http.StatusConflict, err.Error())
		case errors.Is(err, entity.ErrOfferedItemUnavailable):
			api.SendError(c, http.StatusUnprocessableEntity, err.Error())
		default:
			api.SendError(c, http.StatusInternalServerError, "внутренняя ошибка сервера")
		}
		return
	}

	api.SendOk(c, http.StatusOK, newExchangeOfferResponse(updated))
}

// Delete godoc
// @Summary      Delete exchange offer
// @Tags         exchange-offers
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  int  true  "Offer ID"
// @Param        version  query int  true  "Current offer version"
// @Success      204  "No Content"
// @Failure      401  {object}  api.ErrorResponse
// @Failure      403  {object}  api.ErrorResponse  "not the owner"
// @Failure      404  {object}  api.ErrorResponse
// @Failure      409  {object}  api.ErrorResponse  "version conflict / offer locked"
// @Failure      422  {object}  api.ErrorResponse
// @Failure      500  {object}  api.ErrorResponse
// @Router       /exchange-offers/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	userID, ok := api.CurrentUserString(c)
	if !ok {
		return
	}

	requestID, ok := api.PathInt64(c, "id")
	if !ok {
		return
	}

	var query deleteQuery
	if err := validator.BindQuery(&query, c.Request); err != nil {
		api.SendError(c, http.StatusUnprocessableEntity, err.Error())
		return
	}

	if err := h.service.Delete(c.Request.Context(), userID, requestID, query.Version); err != nil {
		var ve validator.Error
		switch {
		case errors.As(err, &ve):
			api.SendError(c, http.StatusUnprocessableEntity, err.Error())
		case errors.Is(err, entity.ErrExchangeOfferNotFound):
			api.SendError(c, http.StatusNotFound, err.Error())
		case errors.Is(err, entity.ErrExchangeOfferForbidden):
			api.SendError(c, http.StatusForbidden, err.Error())
		case errors.Is(err, entity.ErrExchangeOfferVersionConflict),
			errors.Is(err, entity.ErrExchangeOfferLocked):
			api.SendError(c, http.StatusConflict, err.Error())
		default:
			api.SendError(c, http.StatusInternalServerError, "внутренняя ошибка сервера")
		}
		return
	}

	c.Status(http.StatusNoContent)
}
