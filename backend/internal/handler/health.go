package handler

import (
	"database/sql"
	"net/http"
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
	httpStatus := http.StatusOK
	if h.db != nil {
		dbStatus = "ok"
		if err := h.db.Ping(); err != nil {
			dbStatus = "error"
			httpStatus = http.StatusServiceUnavailable
		}
	} else if h.cfg.AppEnv == "production" {
		httpStatus = http.StatusServiceUnavailable
	}

	response.SuccessWithStatus(c, httpStatus, gin.H{
		"app":       h.cfg.AppName,
		"env":       h.cfg.AppEnv,
		"database":  dbStatus,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}
