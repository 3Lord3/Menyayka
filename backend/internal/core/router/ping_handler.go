package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// PingHandler is a scaffold smoke-test endpoint proving the server and routing
// are up. Remove once a real feature exists.
type PingHandler struct{}

func NewPingHandler() *PingHandler {
	return &PingHandler{}
}

// Ping godoc
// @Summary      Smoke-test endpoint
// @Description  Scaffold endpoint proving the server and routing are up.
// @Tags         system
// @Produce      json
// @Success      200  {object}  map[string]string  "{\"status\": \"ok\", \"message\": \"pong\"}"
// @Router       /ping [get]
func (h *PingHandler) Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "pong",
	})
}
