package router

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	"xueju/backend/internal/config"
	"xueju/backend/internal/handler"
	"xueju/backend/internal/middleware"
)

func New(cfg config.Config, db *sql.DB) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Logger(), middleware.Recovery())
	r.NoRoute(middleware.NoRoute)

	api := r.Group("/api")
	healthHandler := handler.NewHealthHandler(cfg, db)
	api.GET("/health", healthHandler.Show)

	adminHandler := handler.NewAdminHandler()
	admin := api.Group("/admin")
	admin.GET("/dashboard", adminHandler.Dashboard)
	admin.GET("/content-reviews", adminHandler.ContentReviews)
	admin.GET("/reports", adminHandler.Reports)
	admin.GET("/users", adminHandler.Users)
	admin.GET("/events", adminHandler.Events)
	admin.GET("/messages", adminHandler.Messages)
	admin.GET("/reviews", adminHandler.Reviews)
	admin.POST("/:resource/:id/actions", adminHandler.Action)

	return r
}
