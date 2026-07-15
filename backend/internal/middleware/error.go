package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"xueju/backend/internal/response"
)

func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, "internal server error")
	})
}

func NoRoute(c *gin.Context) {
	response.Error(c, http.StatusNotFound, response.CodeNotFound, "route not found")
}
