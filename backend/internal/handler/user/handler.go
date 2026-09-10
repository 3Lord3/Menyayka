package user

// Handler serves the "users" feature HTTP endpoints (registration, login, etc.).
type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}
