package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/chasehaye/nas-pipeline/api/internal/handlers"
	"github.com/chasehaye/nas-pipeline/api/internal/store"
)

func RedisRouter(r gin.IRouter, st *store.Store) {
	flights := handlers.NewFlightsHandler(st)
	r.GET("/flights", flights.List)
	r.GET("/flights/:gufi", flights.Get)
}
