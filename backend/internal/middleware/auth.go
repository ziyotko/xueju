package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"xueju/backend/internal/response"
)

const ContextUserID = "userID"
const ContextAdmin = "admin"

func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenText := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		if tokenText == "" {
			response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "missing authorization token")
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenText, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "invalid authorization token")
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			c.Set(ContextUserID, claims["user_id"])
		}
		c.Next()
	}
}

func AdminJWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenText := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		if tokenText == "" {
			response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "missing authorization token")
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenText, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "invalid authorization token")
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || claims["role"] != "admin" {
			response.Error(c, http.StatusForbidden, response.CodeUnauthorized, "admin authorization required")
			c.Abort()
			return
		}
		c.Set(ContextAdmin, true)
		c.Next()
	}
}
