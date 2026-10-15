package exchange_offer

// Handler serves CRUD HTTP requests for exchange requests.
// The user ID comes only from the JWT middleware, never from the request body.
type Handler struct {
	service exchangeOfferService
}

func NewHandler(service exchangeOfferService) *Handler {
	return &Handler{service: service}
}
