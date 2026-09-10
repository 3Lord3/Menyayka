package chain

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Avito-Team-Not-Found/menyayka/internal/api"
)

// Handoff is a temporary local delivery-confirmation callback (without external auth).
// Handoff godoc
// @Summary      Avito handoff callback
// @Description  Temporary local delivery-confirmation callback (no external auth; protect before production).
// @Tags         integrations
// @Accept       json
// @Produce      json
// @Param        body  body      handoffRequest  true  "Handoff payload"
// @Success      200   {object}  fulfillmentResponse
// @Failure      409   {object}  api.ErrorResponse  "chain not ready / handoff pending"
// @Failure      422   {object}  api.ErrorResponse  "invalid payload"
// @Failure      500   {object}  api.ErrorResponse
// @Router       /integrations/avito/handoffs [post]
func (h *Handler) Handoff(c *gin.Context) {
	var body handoffRequest
	if !api.BindJSON(c, &body) {
		return
	}

	result, err := h.service.Handoff(c.Request.Context(), body.ChainID, body.RequestID)
	if err != nil {
		DetermineError(c, err)
		return
	}
	api.SendOk(c, http.StatusOK, result)
}

// ConfirmReceipt godoc
// @Summary      Confirm item receipt
// @Description  Records that the authenticated recipient received the item; the chain completes when every pinned request is DONE.
// @Tags         chains
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int             true  "Chain ID"
// @Param        body  body      receiptRequest  true  "Receipt payload"
// @Success      200   {object}  fulfillmentResponse
// @Failure      401   {object}  api.ErrorResponse
// @Failure      403   {object}  api.ErrorResponse  "not the recipient"
// @Failure      404   {object}  api.ErrorResponse
// @Failure      409   {object}  api.ErrorResponse  "chain/handoff not ready"
// @Failure      422   {object}  api.ErrorResponse  "invalid payload"
// @Failure      500   {object}  api.ErrorResponse
// @Router       /chains/{id}/receipt [post]
func (h *Handler) ConfirmReceipt(c *gin.Context) {
	userID, ok := api.CurrentUserString(c)
	if !ok {
		return
	}
	chainID, ok := api.PathInt64(c, "id")
	if !ok {
		return
	}
	var body receiptRequest
	if !api.BindJSON(c, &body) {
		return
	}

	result, err := h.service.ConfirmReceipt(c.Request.Context(), userID, chainID, body.RequestID)
	if err != nil {
		DetermineError(c, err)
		return
	}
	api.SendOk(c, http.StatusOK, result)
}
