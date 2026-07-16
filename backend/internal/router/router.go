package router

import (
	"database/sql"
	"time"

	"github.com/gin-gonic/gin"
	"xueju/backend/internal/config"
	"xueju/backend/internal/handler"
	"xueju/backend/internal/middleware"
)

func New(cfg config.Config, db *sql.DB) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(middleware.RequestLogger(), middleware.Recovery(), middleware.BodyLimit(6*1024*1024))
	r.NoRoute(middleware.NoRoute)

	api := r.Group("/api")
	healthHandler := handler.NewHealthHandler(cfg, db)
	api.GET("/health", healthHandler.Show)

	adminHandler := handler.NewAdminHandler(cfg, db)
	appHandler := handler.NewAppHandler(cfg, db)
	r.GET("/uploads/:folder/:name", appHandler.ServeUpload)

	api.POST("/auth/wechat-login", appHandler.WechatLogin)
	api.POST("/callbacks/wechat/media", appHandler.MediaReviewCallback)
	api.POST("/admin/auth/login", middleware.RateLimit(10, 5*time.Minute), adminHandler.Login)
	api.GET("/events", appHandler.Events)
	api.GET("/events/:id", appHandler.EventDetail)
	api.GET("/dict/resorts", appHandler.Resorts)
	api.GET("/dict/tags", appHandler.Tags)
	api.GET("/dict/cities", appHandler.Cities)
	api.GET("/users/:id", appHandler.PublicUser)
	api.GET("/users/:id/reviews", appHandler.UserReviews)

	auth := api.Group("")
	auth.Use(middleware.JWTAuth(cfg.JWTSecret), middleware.ActiveUser(db))
	auth.GET("/user/me", appHandler.Me)
	auth.PUT("/user/me", appHandler.UpdateMe)
	auth.DELETE("/user/me", appHandler.DeleteMe)
	auth.GET("/user/verification", appHandler.VerificationStatus)
	auth.POST("/user/verification/sms/send", middleware.RateLimit(30, time.Hour), appHandler.SendPhoneVerificationCode)
	auth.POST("/user/verification/sms/check", middleware.RateLimit(60, time.Hour), appHandler.CheckPhoneVerificationCode)
	auth.POST("/uploads/avatar", appHandler.UploadAvatar)
	auth.POST("/uploads/event-image", appHandler.UploadEventImage)
	auth.GET("/uploads/:id/status", appHandler.UploadStatus)
	auth.POST("/events", appHandler.CreateEvent)
	auth.PUT("/events/:id", appHandler.UpdateEvent)
	auth.DELETE("/events/:id", appHandler.DeleteEvent)
	auth.POST("/events/:id/cancel", appHandler.SetEventStatus("cancelled"))
	auth.POST("/events/:id/finish", appHandler.SetEventStatus("finished"))
	auth.POST("/events/:id/apply", appHandler.ApplyEvent)
	auth.GET("/join-requests/my", appHandler.MyJoinRequests)
	auth.GET("/events/:id/applications", appHandler.EventApplications)
	auth.DELETE("/events/:id/members/:userId", appHandler.RemoveEventMember)
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
	auth.GET("/favorites", appHandler.Favorites)
	auth.POST("/events/:id/favorite", appHandler.SetFavorite(true))
	auth.DELETE("/events/:id/favorite", appHandler.SetFavorite(false))
	auth.POST("/users/:id/follow", appHandler.SetFollow(true))
	auth.DELETE("/users/:id/follow", appHandler.SetFollow(false))
	auth.GET("/users/:id/follow-status", appHandler.FollowStatus)
	auth.GET("/notifications", appHandler.Notifications)
	auth.POST("/notifications/read", appHandler.MarkNotificationsRead)
	auth.DELETE("/notifications", appHandler.ClearNotifications)

	admin := api.Group("/admin")
	admin.Use(middleware.AdminJWTAuth(cfg.JWTSecret))
	admin.GET("/dashboard", adminHandler.Dashboard)
	admin.GET("/moderation/summary", adminHandler.ModerationSummary)
	admin.GET("/content-reviews", adminHandler.ContentReviews)
	admin.GET("/reports", adminHandler.Reports)
	admin.GET("/users", adminHandler.Users)
	admin.GET("/users/:id/verifications", adminHandler.UserVerifications)
	admin.GET("/events", adminHandler.Events)
	admin.GET("/applications", adminHandler.Applications)
	admin.GET("/dicts", adminHandler.Dicts)
	admin.GET("/messages", adminHandler.Messages)
	admin.GET("/reviews", adminHandler.Reviews)
	admin.GET("/uploads", adminHandler.Uploads)
	admin.POST("/uploads/resort-image", adminHandler.UploadResortImage)
	admin.GET("/uploads/:id/preview", adminHandler.UploadPreview)
	admin.GET("/audit-logs", adminHandler.AuditLogs)
	admin.POST("/:resource/:id/actions", adminHandler.Action)
	admin.POST("/dicts", adminHandler.SaveDict)
	admin.PUT("/dicts/:id", adminHandler.SaveDict)

	return r
}
