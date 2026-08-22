package live

import "github.com/gin-gonic/gin"

func Routes(r gin.IRouter, st *Store) {
	h := NewHandler(st)
	r.GET("/flights", h.List)
	r.GET("/flights/:gufi", h.Get)
}
