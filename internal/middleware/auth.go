package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

func APIKeyAuth() gin.HandlerFunc {
	raw := os.Getenv("API_KEYS")
	if raw == "" {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	keys := make(map[string]string, 4)
	for _, pair := range strings.Split(raw, ",") {
		pair = strings.TrimSpace(pair)
		sep := strings.IndexByte(pair, ':')
		if sep < 1 {
			continue
		}
		keys[pair[:sep]] = pair[sep+1:]
	}

	return func(c *gin.Context) {
		key := c.GetHeader("X-API-Key")
		userID, ok := keys[key]
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or missing API key",
			})
			return
		}
		c.Set("user_id", userID)
		c.Next()
	}
}
