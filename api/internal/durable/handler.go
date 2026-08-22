package durable

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	store *Store
}

func NewHandler(st *Store) *Handler {
	return &Handler{store: st}
}

func (h *Handler) GetFlight(c *gin.Context) {
	gufi := c.Param("gufi")
	rec, err := h.store.GetRecord(c.Request.Context(), gufi)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to read flight record"})
		return
	}
	if rec == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "flight not found"})
		return
	}
	c.JSON(http.StatusOK, rec)
}
