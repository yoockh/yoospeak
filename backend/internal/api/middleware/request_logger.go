package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func RequestLogger(l *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		reqID := c.GetHeader("X-Request-Id")
		if reqID == "" {
			reqID = uuid.NewString()
		}
		c.Header("X-Request-Id", reqID)
		c.Set("request_id", reqID)

		c.Next()

		lat := time.Since(start)
		status := c.Writer.Status()

		userID, _ := c.Get("user_id")

		entry := l.WithFields(logrus.Fields{
			"request_id": reqID,
			"method":     c.Request.Method,
			"path":       c.FullPath(),
			"status":     status,
			"latency_ms": lat.Milliseconds(),
			"ip":         c.ClientIP(),
			"user_id":    userID,
		})

		if len(c.Errors) > 0 {
			entry = entry.WithField("errors", c.Errors.String())
		}

		switch {
		case status >= 500:
			entry.Error("request")
		case status >= 400:
			entry.Warn("request")
		default:
			entry.Info("request")
		}
	}
}
