package handler

import (
	"database/sql"
	"time"

	"github.com/gin-gonic/gin"
	"xueju/backend/internal/config"
	"xueju/backend/internal/response"
)

type HealthHandler struct {
	cfg config.Config
	db  *sql.DB
}

func NewHealthHandler(cfg config.Config, db *sql.DB) *HealthHandler {
	return &HealthHandler{cfg: cfg, db: db}
}

func (h *HealthHandler) Show(c *gin.Context) {
	dbStatus := "not_configured"
	if h.db != nil {
		dbStatus = "ok"
		if err := h.db.Ping(); err != nil {
			dbStatus = "error"
		}
	}

	response.Success(c, gin.H{
		"app":       h.cfg.AppName,
		"env":       h.cfg.AppEnv,
		"database":  dbStatus,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

