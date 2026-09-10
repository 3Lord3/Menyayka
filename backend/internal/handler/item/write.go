package item

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Avito-Team-Not-Found/menyayka/internal/api"
	"github.com/Avito-Team-Not-Found/menyayka/internal/entity"

	itemservice "github.com/Avito-Team-Not-Found/menyayka/internal/service/item"
)

// Create godoc
// @Summary      Create item
// @Tags         items
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      createItemRequest  true  "Item payload"
// @Success      201   {object}  itemResponse
// @Failure      401   {object}  api.ErrorResponse
// @Failure      422   {object}  api.ErrorResponse  "validation error"
// @Failure      500   {object}  api.ErrorResponse
// @Router       /items [post]
func (h *Handler) Create(c *gin.Context) {
	ownerID, ok := api.CurrentUserUUID(c)
	if !ok {
		return
	}

	var body createItemRequest
	if !api.BindJSON(c, &body) {
		return
	}

	created, err := h.service.Create(c.Request.Context(), ownerID, itemservice.CreateInput{
		Title:       body.Title,
		Description: body.Description,
		Category:    body.Category,
	})
	if err != nil {
		writeItemError(c, err)
		return
	}

	api.SendOk(c, http.StatusCreated, newItemResponse(created))
}

// Update godoc
// @Summary      Update item
// @Description  Partial update; only the provided fields are changed. Foreign and missing items are both reported as 404.
// @Tags         items
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int                true  "Item ID"
// @Param        body  body      updateItemRequest  true  "Fields to update"
// @Success      200   {object}  itemResponse
// @Failure      401   {object}  api.ErrorResponse
// @Failure      404   {object}  api.ErrorResponse  "not found or not the owner"
// @Failure      409   {object}  api.ErrorResponse  "item has a hard reservation"
// @Failure      422   {object}  api.ErrorResponse  "validation error / archived / invalid status"
// @Failure      500   {object}  api.ErrorResponse
// @Router       /items/{id} [patch]
func (h *Handler) Update(c *gin.Context) {
	userID, ok := api.CurrentUserUUID(c)
	if !ok {
		return
	}

	itemID, ok := api.PathInt64(c, "id")
	if !ok {
		return
	}

	var body updateItemRequest
	if !api.BindJSON(c, &body) {
		return
	}

	updated, err := h.service.Update(c.Request.Context(), userID, itemID, itemservice.UpdateInput{
		Title:       body.Title,
		Description: body.Description,
		Category:    body.Category,
		Status:      body.Status,
	})
	if err != nil {
		writeItemError(c, err)
		return
	}

	api.SendOk(c, http.StatusOK, newItemResponse(updated))
}

// Archive godoc
// @Summary      Archive item
// @Tags         items
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  int  true  "Item ID"
// @Success      204  "No Content"
// @Failure      401  {object}  api.ErrorResponse
// @Failure      404  {object}  api.ErrorResponse  "not found or not the owner"
// @Failure      409  {object}  api.ErrorResponse  "item has a hard reservation"
// @Failure      500  {object}  api.ErrorResponse
// @Router       /items/{id} [delete]
func (h *Handler) Archive(c *gin.Context) {
	userID, ok := api.CurrentUserUUID(c)
	if !ok {
		return
	}

	itemID, ok := api.PathInt64(c, "id")
	if !ok {
		return
	}

	if err := h.service.Archive(c.Request.Context(), userID, itemID); err != nil {
		writeItemError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// UploadImage reads a multipart form field "image" and stores the item photo in
// object storage. Accepts jpeg/png/webp, max size 5 MiB.
// UploadImage godoc
// @Summary      Upload item image
// @Description  Accepts multipart form field "image" (jpeg/png/webp, max 5 MiB) and stores the photo in object storage.
// @Tags         items
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        id     path      int   true  "Item ID"
// @Param        image  formData  file  true  "Image file"
// @Success      200    {object}  itemResponse
// @Failure      401    {object}  api.ErrorResponse
// @Failure      404    {object}  api.ErrorResponse  "not found or not the owner"
// @Failure      422    {object}  api.ErrorResponse  "missing file / unsupported type / too large"
// @Failure      500    {object}  api.ErrorResponse
// @Router       /items/{id}/image [post]
func (h *Handler) UploadImage(c *gin.Context) {
	userID, ok := api.CurrentUserUUID(c)
	if !ok {
		return
	}

	itemID, ok := api.PathInt64(c, "id")
	if !ok {
		return
	}

	fileHeader, err := c.FormFile("image")
	if err != nil {
		api.SendError(c, http.StatusUnprocessableEntity, "необходимо прикрепить изображение")
		return
	}

	if fileHeader.Size > maxImageUploadSize {
		api.SendError(c, http.StatusUnprocessableEntity, entity.ErrImageTooLarge.Error())
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		api.SendError(c, http.StatusUnprocessableEntity, "не удалось прочитать файл изображения")
		return
	}
	defer file.Close()

	contentType := fileHeader.Header.Get("Content-Type")

	updated, err := h.service.UploadImage(c.Request.Context(), userID, itemID, file, fileHeader.Size, contentType)
	if err != nil {
		writeItemError(c, err)
		return
	}

	api.SendOk(c, http.StatusOK, newItemResponse(updated))
}
