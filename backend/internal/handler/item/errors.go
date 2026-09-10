package item

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Avito-Team-Not-Found/menyayka/internal/api"
	"github.com/Avito-Team-Not-Found/menyayka/internal/entity"
	"github.com/Avito-Team-Not-Found/menyayka/pkg/validator"
)

// maxImageUploadSize mirrors the service/item limit so oversized files are rejected
// before being read into memory.
const maxImageUploadSize = 5 << 20

func writeItemError(c *gin.Context, err error) {
	var ve validator.Error
	switch {
	case errors.As(err, &ve):
		api.SendError(c, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, entity.ErrItemNotFound), errors.Is(err, entity.ErrItemForbidden):
		// Foreign and non-existent items are indistinguishable to the client — don't
		// confirm the existence of others' items.
		api.SendError(c, http.StatusNotFound, "товар не найден")
	case errors.Is(err, entity.ErrItemHasHardReservation):
		api.SendError(c, http.StatusConflict, err.Error())
	case errors.Is(err, entity.ErrItemArchived),
		errors.Is(err, entity.ErrInvalidItemStatus),
		errors.Is(err, entity.ErrInvalidImageType),
		errors.Is(err, entity.ErrImageTooLarge):
		api.SendError(c, http.StatusUnprocessableEntity, err.Error())
	default:
		api.SendError(c, http.StatusInternalServerError, "внутренняя ошибка сервера")
	}
}
