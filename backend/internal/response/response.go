package response

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	CodeSuccess      = 0
	CodeBadRequest   = 400
	CodeUnauthorized = 401
	CodeNotFound     = 404
	CodeServerError  = 500
	CodeContentRisk  = 40010
)

type Body struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Body{
		Code:    CodeSuccess,
		Message: "success",
		Data:    data,
	})
}

func Error(c *gin.Context, httpStatus int, code int, message string) {
	if httpStatus >= http.StatusInternalServerError {
		requestID, _ := c.Get("requestID")
		log.Printf("internal request error: requestId=%v status=%d code=%d error=%q", requestID, httpStatus, code, message)
		message = "服务暂时不可用，请稍后重试"
	}
	c.JSON(httpStatus, Body{
		Code:    code,
		Message: message,
		Data:    gin.H{},
	})
}

func ContentRisk(c *gin.Context) {
	Error(c, http.StatusBadRequest, CodeContentRisk, "内容可能包含不适宜信息，请修改后重试")
}
