package middleware

import (
	"fleet-backend/pkg/jwt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)


func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}
		
		tokenString := strings.Replace(authHeader, "Bearer ", "", 1)

		claims, err := jwt.NewJWTUtil().ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return 
		}
		
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)
		c.Next()
	}
}