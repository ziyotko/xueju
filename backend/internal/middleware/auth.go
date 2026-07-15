package middleware

import (
	"database/sql"
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
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
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

func ActiveUser(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if db == nil {
			c.Next()
			return
		}
		value, ok := c.Get(ContextUserID)
		if !ok {
			response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "missing user")
			c.Abort()
			return
		}
		var userID int64
		switch typed := value.(type) {
		case float64:
			userID = int64(typed)
		case int64:
			userID = typed
		case int:
			userID = int64(typed)
		}
		var status string
		if userID == 0 || db.QueryRow(`SELECT status FROM users WHERE id=? AND deleted_at IS NULL`, userID).Scan(&status) != nil {
			response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "user not found")
			c.Abort()
			return
		}
		if status == "disabled" {
			response.Error(c, http.StatusForbidden, response.CodeUnauthorized, "user is disabled")
			c.Abort()
			return
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
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
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
		c.Set("adminUsername", claims["username"])
		c.Next()
	}
}
