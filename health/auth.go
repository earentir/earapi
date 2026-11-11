package health

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// UserCred is one allowed API user.
type UserCred struct {
	Username string
	Token    string
}

// AuthMiddleware validates X-User + token for configured users.
func AuthMiddleware(users []UserCred) gin.HandlerFunc {
	byUser := make(map[string]string, len(users))
	for _, u := range users {
		if u.Username != "" {
			byUser[u.Username] = u.Token
		}
	}
	return func(c *gin.Context) {
		if len(byUser) == 0 {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "health auth not configured",
			})
			return
		}
		user := strings.TrimSpace(c.GetHeader("X-User"))
		if user == "" {
			user = strings.TrimSpace(c.Query("user"))
		}
		token := strings.TrimSpace(c.GetHeader("X-API-Key"))
		if token == "" {
			if h := c.GetHeader("Authorization"); strings.HasPrefix(strings.ToLower(h), "bearer ") {
				token = strings.TrimSpace(h[7:])
			}
		}
		if token == "" {
			token = strings.TrimSpace(c.Query("token"))
		}
		expected, ok := byUser[user]
		if !ok || expected == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		if subtle.ConstantTimeCompare([]byte(token), []byte(expected)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Set("healthUser", user)
		c.Next()
	}
}

// UserFromContext returns the authenticated health username.
func UserFromContext(c *gin.Context) string {
	v, _ := c.Get("healthUser")
	s, _ := v.(string)
	return s
}
