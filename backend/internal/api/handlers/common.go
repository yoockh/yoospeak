package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yoockh/yoospeak/internal/utils"
)

type APIError struct {
	Code    utils.Code `json:"code"`
	Message string     `json:"message"`
}

func writeError(c *gin.Context, err error) {
	status := utils.HTTPStatus(err)

	var ae *utils.AppError
	if errors.As(err, &ae) {
		c.JSON(status, APIError{
			Code:    ae.Code,
			Message: ae.Message,
		})
		return
	}

	c.JSON(status, APIError{
		Code:    utils.CodeInternal,
		Message: http.StatusText(status),
	})
}

func requireUserID(c *gin.Context) (string, bool) {
	if v, ok := c.Get("user_id"); ok {
		if s, ok := v.(string); ok && s != "" {
			return s, true
		}
	}

	writeError(c, utils.E(utils.CodeUnauthorized, "Auth", "unauthorized", nil))
	return "", false
}
