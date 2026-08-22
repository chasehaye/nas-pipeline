package live

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

func (h *Handler) List(c *gin.Context) {
	flights, err := h.store.ListFlights(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to read flights"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": len(flights), "flights": flights})
}

func (h *Handler) Get(c *gin.Context) {
	gufi := c.Param("gufi")
	f, found, err := h.store.GetFlight(c.Request.Context(), gufi)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to read flight"})
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "flight not found"})
		return
	}
	c.JSON(http.StatusOK, f)
}
