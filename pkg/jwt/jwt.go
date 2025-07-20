package jwt

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTUtil struct {
	secretKey []byte
	expiry    time.Duration
}

type Claims struct {
	UserID   string `json:"user_id"`
	Email string `json:"email"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

var globalJWTUtil *JWTUtil

func NewJWTUtil() *JWTUtil {
	if globalJWTUtil != nil {
		return globalJWTUtil
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "default-secret-key-change-this-in-production"
	}

	expiryStr := os.Getenv("JWT_EXPIRY")
	expiry, err := time.ParseDuration(expiryStr)
	if err != nil {
		expiry = 24 * time.Hour
	}

	// fmt.Printf("JWT Config - Secret: %s, Expiry: %s\n", secret[:10]+"...", expiry.String())

	globalJWTUtil = &JWTUtil{
		secretKey: []byte(secret),
		expiry:    expiry,
	}

	return globalJWTUtil
}

func (j *JWTUtil) GenerateToken(userID, email, role string) (string, error) {
	now := time.Now()
	expiresAt := now.Add(j.expiry)
	
	claims := &Claims{
		UserID:   userID,
		Email: email,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "fleet-management-system",
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(j.secretKey)
	
	if err == nil {
		fmt.Printf("JWT Debug - Generated token for UserID=%s, Email=%s, Role=%s, ExpiresAt=%s\n", 
			userID, email, role, expiresAt.Format(time.RFC3339))
		fmt.Printf("JWT Debug - Token: %s\n", tokenString[:min(50, len(tokenString))]+"...")
	}
	
	return tokenString, err
}

func (j *JWTUtil) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return j.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

func (j *JWTUtil) RefreshToken(tokenString string) (string, error) {
	// First try normal validation
	claims, err := j.ValidateToken(tokenString)
	if err != nil {
		// If validation fails, try parsing without validation to check if it's just expired
		claims, err = j.parseExpiredToken(tokenString)
		if err != nil {
			return "", err
		}
		
		// Check if token expired within grace period (24 hours)
		gracePeriod := 24 * time.Hour
		if time.Since(claims.ExpiresAt.Time) > gracePeriod {
			return "", errors.New("token expired beyond grace period")
		}
	} else {
		// Token is still valid, check if it needs refresh (within 1 hour of expiry)
		if time.Until(claims.ExpiresAt.Time) > time.Hour {
			return tokenString, nil // Token is still valid for more than 1 hour
		}
	}

	// Generate new token
	return j.GenerateToken(claims.UserID, claims.Email, claims.Role)
}

// parseExpiredToken parses a token without validating expiration
func (j *JWTUtil) parseExpiredToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return j.secretKey, nil
	}, jwt.WithoutClaimsValidation())

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok {
		return claims, nil
	}

	return nil, errors.New("invalid token claims")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}