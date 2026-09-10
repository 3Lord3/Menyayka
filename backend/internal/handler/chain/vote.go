package chain

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Avito-Team-Not-Found/menyayka/internal/api"
	"github.com/Avito-Team-Not-Found/menyayka/internal/entity"
	chainservice "github.com/Avito-Team-Not-Found/menyayka/internal/service/chain"
	"github.com/Avito-Team-Not-Found/menyayka/pkg/validator"
)

// Vote godoc
// @Summary      Vote for an exchange option
// @Tags         chains
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int          true  "Chain ID"
// @Param        body  body      voteRequest  true  "Vote payload"
// @Success      200   {object}  voteResponse
// @Failure      401   {object}  api.ErrorResponse
// @Failure      403   {object}  api.ErrorResponse  "vote not allowed"
// @Failure      404   {object}  api.ErrorResponse
// @Failure      409   {object}  api.ErrorResponse  "not a candidate"
// @Failure      422   {object}  api.ErrorResponse  "validation error / invalid target"
// @Failure      500   {object}  api.ErrorResponse
// @Router       /chains/{id}/votes [put]
func (h *Handler) Vote(c *gin.Context) {
	userID, ok := api.CurrentUserString(c)
	if !ok {
		return
	}

	chainID, ok := api.PathInt64(c, "id")
	if !ok {
		return
	}

	var body voteRequest
	if !api.BindJSON(c, &body) {
		return
	}

	vote, err := h.service.Vote(c.Request.Context(), userID, chainID, chainservice.VoteInput{
		RequestID:       body.RequestID,
		TargetRequestID: body.TargetRequestID,
	})
	if err != nil {
		var ve validator.Error
		switch {
		case errors.As(err, &ve):
			api.SendError(c, http.StatusUnprocessableEntity, err.Error())
		case errors.Is(err, entity.ErrChainNotFound):
			api.SendError(c, http.StatusNotFound, err.Error())
		case errors.Is(err, entity.ErrChainVoteForbidden):
			api.SendError(c, http.StatusForbidden, err.Error())
		case errors.Is(err, entity.ErrChainNotCandidate):
			api.SendError(c, http.StatusConflict, err.Error())
		case errors.Is(err, entity.ErrInvalidVoteTarget):
			api.SendError(c, http.StatusUnprocessableEntity, err.Error())
		default:
			api.SendError(c, http.StatusInternalServerError, "внутренняя ошибка сервера")
		}
		return
	}

	api.SendOk(c, http.StatusOK, voteResponse{
		ChainID:         vote.ChainID,
		RequestID:       vote.RequestID,
		TargetRequestID: vote.TargetRequestID,
		Vote:            vote.Vote,
		VotedAt:         vote.VotedAt.UTC().Format(time.RFC3339),
		ChainStatus:     vote.ChainStatus,
	})
}

// WithdrawVote godoc
// @Summary      Withdraw a vote
// @Tags         chains
// @Produce      json
// @Security     BearerAuth
// @Param        id               path   int  true  "Chain ID"
// @Param        requestId        query  int  true  "Request ID"
// @Param        targetRequestId  query  int  true  "Target request ID"
// @Success      204  "No Content"
// @Failure      401  {object}  api.ErrorResponse
// @Failure      403  {object}  api.ErrorResponse
// @Failure      404  {object}  api.ErrorResponse
// @Failure      409  {object}  api.ErrorResponse
// @Failure      422  {object}  api.ErrorResponse
// @Failure      500  {object}  api.ErrorResponse
// @Router       /chains/{id}/votes [delete]
func (h *Handler) WithdrawVote(c *gin.Context) {
	userID, ok := api.CurrentUserString(c)
	if !ok {
		return
	}

	chainID, ok := api.PathInt64(c, "id")
	if !ok {
		return
	}
	var query withdrawVoteQuery
	if err := validator.BindQuery(&query, c.Request); err != nil {
		api.SendError(c, http.StatusUnprocessableEntity, err.Error())
		return
	}

	err := h.service.WithdrawVote(c.Request.Context(), userID, chainID, chainservice.VoteInput{
		RequestID:       query.RequestID,
		TargetRequestID: query.TargetRequestID,
	})
	if err != nil {
		var ve validator.Error
		switch {
		case errors.As(err, &ve):
			api.SendError(c, http.StatusUnprocessableEntity, err.Error())
		case errors.Is(err, entity.ErrChainNotFound):
			api.SendError(c, http.StatusNotFound, err.Error())
		case errors.Is(err, entity.ErrChainVoteForbidden):
			api.SendError(c, http.StatusForbidden, err.Error())
		case errors.Is(err, entity.ErrChainNotCandidate):
			api.SendError(c, http.StatusConflict, err.Error())
		case errors.Is(err, entity.ErrInvalidVoteTarget):
			api.SendError(c, http.StatusUnprocessableEntity, err.Error())
		default:
			api.SendError(c, http.StatusInternalServerError, "внутренняя ошибка сервера")
		}
		return
	}

	c.Status(http.StatusNoContent)
}

// Confirm godoc
// @Summary      Confirm participation in a chain
// @Tags         chains
// @Produce      json
// @Security     BearerAuth
// @Param        id  path      int  true  "Chain ID"
// @Success      200  {object}  confirmResponse
// @Failure      401  {object}  api.ErrorResponse
// @Failure      403  {object}  api.ErrorResponse
// @Failure      404  {object}  api.ErrorResponse
// @Failure      409  {object}  api.ErrorResponse  "chain not proposed / request in two frozen chains"
// @Failure      410  {object}  api.ErrorResponse  "confirmation expired"
// @Failure      500  {object}  api.ErrorResponse
// @Router       /chains/{id}/confirm [post]
func (h *Handler) Confirm(c *gin.Context) {
	userID, ok := api.CurrentUserString(c)
	if !ok {
		return
	}

	chainID, ok := api.PathInt64(c, "id")
	if !ok {
		return
	}

	status, err := h.service.Confirm(c.Request.Context(), userID, chainID)
	if err != nil {
		switch {
		case errors.Is(err, entity.ErrChainNotFound):
			api.SendError(c, http.StatusNotFound, err.Error())
		case errors.Is(err, entity.ErrChainNotProposed):
			api.SendError(c, http.StatusConflict, err.Error())
		case errors.Is(err, entity.ErrRequestInTwoFrozenChains):
			api.SendError(c, http.StatusConflict, err.Error())
		case errors.Is(err, entity.ErrChainVoteForbidden):
			api.SendError(c, http.StatusForbidden, err.Error())
		case errors.Is(err, entity.ErrChainConfirmationExpired):
			api.SendError(c, http.StatusGone, err.Error())
		default:
			log.Printf("confirm chain failed: %v", err)
			api.SendError(c, http.StatusInternalServerError, "внутренняя ошибка сервера")
		}
		return
	}

	api.SendOk(c, http.StatusOK, gin.H{"chainId": chainID, "status": status})
}

// Think godoc
// @Summary      Mark an option as "thinking"
// @Tags         chains
// @Produce      json
// @Security     BearerAuth
// @Param        id  path      int  true  "Chain ID"
// @Success      200  {object}  thinkResponse
// @Failure      401  {object}  api.ErrorResponse
// @Failure      403  {object}  api.ErrorResponse
// @Failure      404  {object}  api.ErrorResponse
// @Failure      409  {object}  api.ErrorResponse
// @Failure      410  {object}  api.ErrorResponse
// @Failure      500  {object}  api.ErrorResponse
// @Router       /chains/{id}/think [post]
func (h *Handler) Think(c *gin.Context) {
	userID, ok := api.CurrentUserString(c)
	if !ok {
		return
	}
	chainID, ok := api.PathInt64(c, "id")
	if !ok {
		return
	}
	if err := h.service.Think(c.Request.Context(), userID, chainID); err != nil {
		DetermineError(c, err)
		return
	}
	api.SendOk(c, http.StatusOK, gin.H{"chainId": chainID, "vote": entity.VoteThinking})
}

// Unconfirm returns the caller's round-two decision to pending. It does not
// remove the candidate response, so the participant can confirm again later.
// Unconfirm godoc
// @Summary      Undo a round-two confirmation
// @Description  Returns the caller's decision to pending without removing the candidate response.
// @Tags         chains
// @Produce      json
// @Security     BearerAuth
// @Param        id  path      int  true  "Chain ID"
// @Success      200  {object}  unconfirmResponse
// @Failure      401  {object}  api.ErrorResponse
// @Failure      403  {object}  api.ErrorResponse
// @Failure      404  {object}  api.ErrorResponse
// @Failure      409  {object}  api.ErrorResponse
// @Failure      410  {object}  api.ErrorResponse
// @Failure      500  {object}  api.ErrorResponse
// @Router       /chains/{id}/unconfirm [post]
func (h *Handler) Unconfirm(c *gin.Context) {
	userID, ok := api.CurrentUserString(c)
	if !ok {
		return
	}
	chainID, ok := api.PathInt64(c, "id")
	if !ok {
		return
	}
	status, err := h.service.Unconfirm(c.Request.Context(), userID, chainID)
	if err != nil {
		DetermineError(c, err)
		return
	}
	api.SendOk(c, http.StatusOK, gin.H{"chainId": chainID, "status": status, "vote": entity.VotePending})
}

// Decline rejects a chain; the response flags whether a quick replacement is available.
// Decline godoc
// @Summary      Decline a chain
// @Description  Rejects a chain; the response flags whether a quick replacement is available.
// @Tags         chains
// @Produce      json
// @Security     BearerAuth
// @Param        id  path      int  true  "Chain ID"
// @Success      200  {object}  declineResponse
// @Failure      401  {object}  api.ErrorResponse
// @Failure      403  {object}  api.ErrorResponse
// @Failure      404  {object}  api.ErrorResponse
// @Failure      409  {object}  api.ErrorResponse
// @Failure      410  {object}  api.ErrorResponse
// @Failure      500  {object}  api.ErrorResponse
// @Router       /chains/{id}/decline [post]
func (h *Handler) Decline(c *gin.Context) {
	userID, ok := api.CurrentUserString(c)
	if !ok {
		return
	}
	chainID, ok := api.PathInt64(c, "id")
	if !ok {
		return
	}
	replacementAvailable, status, err := h.service.Decline(c.Request.Context(), userID, chainID)
	if err != nil {
		DetermineError(c, err)
		return
	}
	api.SendOk(c, http.StatusOK, gin.H{
		"chainId": chainID, "status": status, "replacementAvailable": replacementAvailable,
	})
}

// SelectReplacement godoc
// @Summary      Select a replacement
// @Tags         chains
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int                       true  "Chain ID"
// @Param        body  body      selectReplacementRequest  true  "Replacement payload"
// @Success      200   {object}  selectReplacementResponse
// @Failure      401   {object}  api.ErrorResponse
// @Failure      403   {object}  api.ErrorResponse
// @Failure      404   {object}  api.ErrorResponse
// @Failure      409   {object}  api.ErrorResponse
// @Failure      422   {object}  api.ErrorResponse  "invalid replacement target"
// @Failure      500   {object}  api.ErrorResponse
// @Router       /chains/{id}/replacement [put]
func (h *Handler) SelectReplacement(c *gin.Context) {
	userID, ok := api.CurrentUserString(c)
	if !ok {
		return
	}
	chainID, ok := api.PathInt64(c, "id")
	if !ok {
		return
	}
	var body selectReplacementRequest
	if !api.BindJSON(c, &body) {
		return
	}
	if err := h.service.SelectReplacement(c.Request.Context(), userID, chainID, body.RequestID); err != nil {
		if errors.Is(err, entity.ErrInvalidVoteTarget) {
			api.SendError(c, http.StatusUnprocessableEntity, err.Error())
			return
		}
		DetermineError(c, err)
		return
	}
	api.SendOk(c, http.StatusOK, gin.H{"chainId": chainID, "requestId": body.RequestID, "status": entity.ChainStatusProposed})
}
