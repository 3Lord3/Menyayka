package item

// Handler serves CRUD HTTP requests for items.
// The owner comes only from the JWT middleware, never from the request body.
type Handler struct {
	service itemService
}

func NewHandler(service itemService) *Handler {
	return &Handler{service: service}
}
