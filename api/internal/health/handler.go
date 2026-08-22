package health

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Pinger is anything whose reachability represents service health (the live
// Redis store).
type Pinger interface {
	Ping(ctx context.Context) error
}

type Handler struct {
	pinger Pinger
}

func NewHandler(p Pinger) *Handler {
	return &Handler{pinger: p}
}

func (h *Handler) Healthz(c *gin.Context) {
	if err := h.pinger.Ping(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "degraded", "redis": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
