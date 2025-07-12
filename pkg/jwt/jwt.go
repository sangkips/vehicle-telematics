package jwt

import (
	"errors"
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

func NewJWTUtil() *JWTUtil {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "default-secret-key-change-this-in-production"
	}

	expiryStr := os.Getenv("JWT_EXPIRY")
	expiry, err := time.ParseDuration(expiryStr)
	if err != nil {
		expiry = 24 * time.Hour
	}

	return &JWTUtil{
		secretKey: []byte(secret),
		expiry:    expiry,
	}
}

func (j *JWTUtil) GenerateToken(userID, email, role string) (string, error) {
	claims := &Claims{
		UserID:   userID,
		Email: email,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "fleet-management-system",
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secretKey)
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
	claims, err := j.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}

	// Check if token is close to expiry (within 1 hour)
	if time.Until(claims.ExpiresAt.Time) > time.Hour {
		return tokenString, nil // Token is still valid for more than 1 hour
	}

	// Generate new token
	return j.GenerateToken(claims.UserID, claims.Email, claims.Role)
}