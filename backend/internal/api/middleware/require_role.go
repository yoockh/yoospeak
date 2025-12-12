package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yoockh/yoospeak/internal/utils"
)

func RequireRole(allowed ...string) gin.HandlerFunc {
	allow := map[string]struct{}{}
	for _, a := range allowed {
		a = strings.TrimSpace(strings.ToLower(a))
		if a != "" {
			allow[a] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		v, ok := c.Get("role")
		role, _ := v.(string)
		role = strings.ToLower(strings.TrimSpace(role))

		if !ok || role == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    utils.CodeForbidden,
				"message": "forbidden",
			})
			return
		}

		if _, ok := allow[role]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    utils.CodeForbidden,
				"message": "forbidden",
			})
			return
		}

		c.Next()
	}
}

func RequireAdmin() gin.HandlerFunc { return RequireRole("admin") }
