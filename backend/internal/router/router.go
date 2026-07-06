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
	r.Static("/uploads", "./uploads")

	api := r.Group("/api")
	healthHandler := handler.NewHealthHandler(cfg, db)
	api.GET("/health", healthHandler.Show)

	adminHandler := handler.NewAdminHandler(db)
	appHandler := handler.NewAppHandler(cfg, db)

	api.POST("/auth/wechat-login", appHandler.WechatLogin)
	api.GET("/events", appHandler.Events)
	api.GET("/events/:id", appHandler.EventDetail)
	api.GET("/dict/resorts", appHandler.Resorts)
	api.GET("/dict/tags", appHandler.Tags)
	api.GET("/dict/cities", appHandler.Cities)
	api.GET("/users/:id/reviews", appHandler.UserReviews)

	auth := api.Group("")
	auth.Use(middleware.JWTAuth(cfg.JWTSecret))
	auth.GET("/user/me", appHandler.Me)
	auth.PUT("/user/me", appHandler.UpdateMe)
	auth.POST("/uploads/avatar", appHandler.UploadAvatar)
	auth.POST("/uploads/event-image", appHandler.UploadEventImage)
	auth.POST("/events", appHandler.CreateEvent)
	auth.PUT("/events/:id", appHandler.UpdateEvent)
	auth.DELETE("/events/:id", appHandler.DeleteEvent)
	auth.POST("/events/:id/cancel", appHandler.SetEventStatus("cancelled"))
	auth.POST("/events/:id/finish", appHandler.SetEventStatus("finished"))
	auth.POST("/events/:id/apply", appHandler.ApplyEvent)
	auth.GET("/join-requests/my", appHandler.MyJoinRequests)
	auth.GET("/events/:id/applications", appHandler.EventApplications)
	auth.POST("/join-requests/:id/approve", appHandler.ReviewJoinRequest("approved"))
	auth.POST("/join-requests/:id/reject", appHandler.ReviewJoinRequest("rejected"))
	auth.GET("/trips/created", appHandler.Trips("created"))
	auth.GET("/trips/joined", appHandler.Trips("joined"))
	auth.GET("/trips/pending", appHandler.Trips("pending"))
	auth.GET("/trips/finished", appHandler.Trips("finished"))
	auth.GET("/chat/conversations", appHandler.ChatConversations)
	auth.POST("/chat/conversations/read", appHandler.MarkAllChatsRead)
	auth.GET("/events/:id/messages", appHandler.Messages)
	auth.POST("/events/:id/messages", appHandler.SendMessage)
	auth.POST("/events/:id/messages/read", appHandler.MarkChatRead)
	auth.POST("/reviews", appHandler.CreateReview)
	auth.POST("/reports", appHandler.CreateReport)

	admin := api.Group("/admin")
	admin.GET("/dashboard", adminHandler.Dashboard)
	admin.GET("/content-reviews", adminHandler.ContentReviews)
	admin.GET("/reports", adminHandler.Reports)
	admin.GET("/users", adminHandler.Users)
	admin.GET("/events", adminHandler.Events)
	admin.GET("/applications", adminHandler.Applications)
	admin.GET("/dicts", adminHandler.Dicts)
	admin.GET("/messages", adminHandler.Messages)
	admin.GET("/reviews", adminHandler.Reviews)
	admin.POST("/:resource/:id/actions", adminHandler.Action)

	return r
}
