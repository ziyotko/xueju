package handler

import (
	"github.com/gin-gonic/gin"
	"xueju/backend/internal/compliance"
	"xueju/backend/internal/response"
)

type AdminHandler struct{}

func NewAdminHandler() *AdminHandler {
	return &AdminHandler{}
}

func (h *AdminHandler) Dashboard(c *gin.Context) {
	response.Success(c, gin.H{
		"contentReview": gin.H{
			"pending":  0,
			"approved": 0,
			"rejected": 0,
		},
		"reports": gin.H{
			"pending": 0,
			"handled": 0,
		},
		"actions": []compliance.AdminAction{
			compliance.ActionDisableUser,
			compliance.ActionDelistEvent,
			compliance.ActionHideMessage,
			compliance.ActionHideReview,
			compliance.ActionResolveReport,
		},
	})
}

func (h *AdminHandler) ContentReviews(c *gin.Context) {
	response.Success(c, gin.H{
		"list":     []interface{}{},
		"page":     1,
		"pageSize": 10,
		"total":    0,
	})
}

func (h *AdminHandler) Reports(c *gin.Context) {
	h.emptyPage(c)
}

func (h *AdminHandler) Users(c *gin.Context) {
	h.emptyPage(c)
}

func (h *AdminHandler) Events(c *gin.Context) {
	h.emptyPage(c)
}

func (h *AdminHandler) Messages(c *gin.Context) {
	h.emptyPage(c)
}

func (h *AdminHandler) Reviews(c *gin.Context) {
	h.emptyPage(c)
}

func (h *AdminHandler) Action(c *gin.Context) {
	response.Success(c, gin.H{
		"status": "accepted",
	})
}

func (h *AdminHandler) emptyPage(c *gin.Context) {
	response.Success(c, gin.H{
		"list":     []interface{}{},
		"page":     1,
		"pageSize": 10,
		"total":    0,
	})
}
