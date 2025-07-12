package services

import (
	"errors"
	"fleet-backend/internal/models"
	"fleet-backend/internal/repository"
	"fleet-backend/pkg/jwt"
	"time"

	"golang.org/x/crypto/bcrypt"
)


type AuthService struct {
	userRepo *repository.UserRepository
	jwtUtil  *jwt.JWTUtil
}

func NewAuthService(userRepo *repository.UserRepository) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		jwtUtil:  jwt.NewJWTUtil(),
	}
}

type LoginRequest struct {
	// Username 	string `json:"username" validate:"required"`
	Email 		string `json:"email" validate:"required"`
	Password 	string `json:"password" validate:"required"`
}

type LoginResponse struct {
	User  *models.AuthUser `json:"user"`
	Token string           `json:"token"`
}

func (s *AuthService) Login(req *LoginRequest) (*LoginResponse, error) {
	// Find user by username
	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Check if user is active
	if user.Status != "active" {
		return nil, errors.New("account is not active")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Update last login
	user.LastLogin = &time.Time{}
	*user.LastLogin = time.Now()
	s.userRepo.Update(user.ID.Hex(), user)

	// Generate JWT token
	token, err := s.jwtUtil.GenerateToken(user.ID.Hex(), user.Email, user.Role)
	if err != nil {
		return nil, errors.New("failed to generate token")
	}

	// Create auth user response
	// authUser := &models.AuthUser{
	// 	ID:          user.ID.Hex(),
	// 	Username:    user.Username,
	// 	Email:       user.Email,
	// 	FirstName:   user.FirstName,
	// 	LastName:    user.LastName,
	// 	Role:        user.Role,
	// 	Permissions: user.Permissions,
	// }

	return &LoginResponse{
		// User:  authUser,
		Token: token,
	}, nil
}

func (s *AuthService) RefreshToken(userID string) (string, error) {
	// Find user by ID
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return "", errors.New("user not found")
	}

	// Check if user is still active
	if user.Status != "active" {
		return "", errors.New("account is not active")
	}

	// Generate new token
	token, err := s.jwtUtil.GenerateToken(user.ID.Hex(), user.Username, user.Role)
	if err != nil {
		return "", errors.New("failed to generate token")
	}

	return token, nil
}

func (s *AuthService) ValidateToken(tokenString string) (*models.AuthUser, error) {
	claims, err := s.jwtUtil.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	// Find user to get latest info
	user, err := s.userRepo.FindByID(claims.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	if user.Status != "active" {
		return nil, errors.New("account is not active")
	}

	return &models.AuthUser{
		ID:          user.ID.Hex(),
		Username:    user.Username,
		Email:       user.Email,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		Role:        user.Role,
		Permissions: user.Permissions,
	}, nil
}

func (s *AuthService) HashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}