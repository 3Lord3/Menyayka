package item

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Avito-Team-Not-Found/menyayka/internal/api"
	"github.com/Avito-Team-Not-Found/menyayka/pkg/validator"

	itemservice "github.com/Avito-Team-Not-Found/menyayka/internal/service/item"
)

// Get godoc
// @Summary      Get item
// @Tags         items
// @Produce      json
// @Security     BearerAuth
// @Param        id  path      int  true  "Item ID"
// @Success      200  {object}  itemResponse
// @Failure      401  {object}  api.ErrorResponse
// @Failure      404  {object}  api.ErrorResponse  "not found or not the owner"
// @Failure      500  {object}  api.ErrorResponse
// @Router       /items/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	userID, ok := api.CurrentUserUUID(c)
	if !ok {
		return
	}

	itemID, ok := api.PathInt64(c, "id")
	if !ok {
		return
	}

	found, err := h.service.Get(c.Request.Context(), userID, itemID)
	if err != nil {
		writeItemError(c, err)
		return
	}

	api.SendOk(c, http.StatusOK, newItemResponse(found))
}

// List godoc
// @Summary      List my items
// @Tags         items
// @Produce      json
// @Security     BearerAuth
// @Param        page      query     int  false  "Page (default 1)"
// @Param        pageSize  query     int  false  "Page size (default 20)"
// @Success      200       {object}  listItemsResponse
// @Failure      401       {object}  api.ErrorResponse
// @Failure      422       {object}  api.ErrorResponse  "validation error"
// @Failure      500       {object}  api.ErrorResponse
// @Router       /items [get]
func (h *Handler) List(c *gin.Context) {
	userID, ok := api.CurrentUserUUID(c)
	if !ok {
		return
	}

	var query listItemsQuery
	if err := validator.BindQuery(&query, c.Request); err != nil {
		api.SendError(c, http.StatusUnprocessableEntity, err.Error())
		return
	}
	page, pageSize := itemservice.NormalizePagination(query.Page, query.PageSize)

	items, total, err := h.service.List(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		api.SendError(c, http.StatusInternalServerError, "внутренняя ошибка сервера")
		return
	}

	response := listItemsResponse{
		Items:    make([]itemResponse, 0, len(items)),
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	}
	for _, it := range items {
		response.Items = append(response.Items, newItemResponse(it))
	}

	api.SendOk(c, http.StatusOK, response)
}
