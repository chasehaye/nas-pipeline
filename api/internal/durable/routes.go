package durable

import "github.com/gin-gonic/gin"

func Routes(r gin.IRouter, st *Store) {
	h := NewHandler(st)
	g := r.Group("/durable")
	g.GET("/flights/:gufi", h.GetFlight)
}
