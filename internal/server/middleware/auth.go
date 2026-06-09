package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/internal/server/service"
)

func AuthRequired(i do.Injector) gin.HandlerFunc {
	tokenValidator := do.MustInvoke[*service.TokenValidator](i)

	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/v1/auth") {
			c.Next()
			return
		}

		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{"error": "missing or invalid token"},
			)
			return
		}

		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		userID, err := tokenValidator.ValidateUserToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{"error": "invalid token"},
			)
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}
