package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type successResponse struct {
	Code uint        `json:"code"`
	Data interface{} `json:"data"`
}

type errorResponse struct {
	Code    uint   `json:"code"`
	Message string `json:"message"`
}

func SuccessResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, successResponse{
		Code: http.StatusOK,
		Data: data,
	})
}

func BadRequestResponse(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, errorResponse{
		Code:    http.StatusBadRequest,
		Message: message,
	})
}

func UnprocessableEntityResponse(c *gin.Context, message string) {
	c.JSON(http.StatusUnprocessableEntity, errorResponse{
		Code:    http.StatusUnprocessableEntity,
		Message: message,
	})
}

func UnauthorizedResponse(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, errorResponse{
		Code:    http.StatusUnauthorized,
		Message: message,
	})
}

func ForbiddenResponse(c *gin.Context, message string) {
	c.JSON(http.StatusForbidden, errorResponse{
		Code:    http.StatusForbidden,
		Message: message,
	})
}

func NotFoundResponse(c *gin.Context, message string) {
	c.JSON(http.StatusNotFound, errorResponse{
		Code:    http.StatusNotFound,
		Message: message,
	})
}

func InternalServerErrorResponse(c *gin.Context, message string) {
	c.JSON(http.StatusInternalServerError, errorResponse{
		Code:    http.StatusInternalServerError,
		Message: message,
	})
}
