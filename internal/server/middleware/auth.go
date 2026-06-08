package middleware

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"

	jwtutil "github.com/minhnbnt/uptime-monitor/internal/server/infrastructure/jwt"
)

func getTokenUserID(c *gin.Context, parser *jwtutil.JwtParser) (uint, error) {

	auth := c.GetHeader("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return 0, errors.New("missing or invalid token")
	}

	tokenStr := strings.TrimPrefix(auth, "Bearer ")
	token, err := parser.Parse(tokenStr)
	if err != nil {
		return 0, errors.New("invalid token")
	}

	sub, err := token.Subject()
	if err != nil {
		return 0, errors.New("invalid token claims")
	}

	userID, err := strconv.ParseUint(sub, 10, 64)
	if err != nil {
		return 0, errors.New("invalid token claims")
	}

	return uint(userID), nil
}

func AuthRequired(i do.Injector) gin.HandlerFunc {
	parser := do.MustInvoke[*jwtutil.JwtParser](i)

	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/v1/auth") {
			c.Next()
			return
		}

		userID, err := getTokenUserID(c, parser)
		if err != nil {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{"error": err.Error()},
			)
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}
