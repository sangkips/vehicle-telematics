package middleware

import (
	"fleet-backend/pkg/jwt"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

var jwtUtil *jwt.JWTUtil

func init() {
	jwtUtil = jwt.NewJWTUtil()
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}
		
		// Handle both "Bearer token" and just "token" formats
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			// No "Bearer " prefix found, use the header as-is
			tokenString = authHeader
		}


		claims, err := jwtUtil.ValidateToken(tokenString)
		if err != nil {
			fmt.Printf("Auth Debug - Validation Error: %v\n", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token", "debug": err.Error()})
			c.Abort()
			return 
		}
		
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}