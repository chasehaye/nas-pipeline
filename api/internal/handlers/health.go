package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/chasehaye/nas-pipeline/api/internal/store"
)


type HealthHandler struct {
	store *store.Store
}

func NewHealthHandler(st *store.Store) *HealthHandler {
	return &HealthHandler{store: st}
}

func (h *HealthHandler) Healthz(c *gin.Context) {
	if err := h.store.Ping(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "degraded", "redis": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
