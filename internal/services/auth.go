package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fleet-backend/internal/models"
	"fleet-backend/internal/repository"
	"fleet-backend/pkg/email"
	"fleet-backend/pkg/jwt"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo     *repository.UserRepository
	jwtUtil      *jwt.JWTUtil
	emailService *email.EmailService
}

func NewAuthService(userRepo *repository.UserRepository, emailService *email.EmailService) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		jwtUtil:      jwt.NewJWTUtil(),
		emailService: emailService,
	}
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type LoginResponse struct {
	User  *models.AuthUser `json:"user"`
	Token string           `json:"token"`
}

func (s *AuthService) Login(req *LoginRequest) (*LoginResponse, error) {
	// Find user by email
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
	authUser := &models.AuthUser{
		ID:          user.ID.Hex(),
		Username:    user.Username,
		Email:       user.Email,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		Role:        user.Role,
		Permissions: user.Permissions,
	}

	return &LoginResponse{
		User:  authUser,
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
	token, err := s.jwtUtil.GenerateToken(user.ID.Hex(), user.Email, user.Role)
	if err != nil {
		return "", errors.New("failed to generate token")
	}

	return token, nil
}

func (s *AuthService) RefreshTokenFromString(tokenString string) (string, error) {
	// Use the JWT util's built-in refresh logic
	newToken, err := s.jwtUtil.RefreshToken(tokenString)
	if err != nil {
		return "", errors.New("failed to refresh token")
	}

	return newToken, nil
}

func (s *AuthService) GetUserProfile(userID string) (*models.AuthUser, error) {
	// Find user by ID
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Check if user is still active
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

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// ForgotPassword initiates the password reset process
func (s *AuthService) ForgotPassword(email string) error {
	// Find user by email (but don't reveal if user exists for security)
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		// Return success even if user doesn't exist (prevent email enumeration)
		// But log it for debugging
		fmt.Printf("DEBUG: User not found for email: %s, error: %v\n", email, err)
		return nil
	}

	// Generate secure random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		fmt.Printf("ERROR: Failed to generate reset token: %v\n", err)
		return errors.New("failed to generate reset token")
	}
	token := hex.EncodeToString(tokenBytes)

	// Hash the token before storing
	hashedToken, err := bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
	if err != nil {
		fmt.Printf("ERROR: Failed to hash reset token: %v\n", err)
		return errors.New("failed to hash reset token")
	}

	// Set expiry to 24 hours from now
	expiry := time.Now().Add(24 * time.Hour)

	// Update user with reset token
	if err := s.userRepo.UpdatePasswordResetToken(email, string(hashedToken), expiry); err != nil {
		fmt.Printf("ERROR: Failed to update reset token in database: %v\n", err)
		return errors.New("failed to update reset token")
	}

	fmt.Printf("DEBUG: Updated database with reset token\n")

	// Send reset email
	if err := s.emailService.SendPasswordResetEmail(user.Email, token); err != nil {
		// Log error but don't fail the request
		fmt.Printf("ERROR: Failed to send reset email: %v\n", err)
		return errors.New("failed to send reset email")
	}

	fmt.Printf("SUCCESS: Password reset email sent to %s\n", user.Email)
	return nil
}

type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"newPassword" validate:"required,min=6"`
}

// ResetPassword resets a user's password using a valid reset token
func (s *AuthService) ResetPassword(token, newPassword string) error {
	// Find all users with non-expired reset tokens
	// We need to check each one because tokens are hashed
	users, err := s.userRepo.FindAll()
	if err != nil {
		return errors.New("failed to process reset request")
	}

	var matchedUser *models.User
	for _, user := range users {
		if user.PasswordResetToken == "" || user.PasswordResetExpiry == nil {
			continue
		}

		// Check if token is expired
		if user.PasswordResetExpiry.Before(time.Now()) {
			continue
		}

		// Compare hashed token
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordResetToken), []byte(token)); err == nil {
			matchedUser = user
			break
		}
	}

	if matchedUser == nil {
		return errors.New("invalid or expired reset token")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("failed to hash new password")
	}

	// Update user password
	matchedUser.Password = string(hashedPassword)
	matchedUser.UpdatedAt = time.Now()

	if _, err := s.userRepo.Update(matchedUser.ID.Hex(), matchedUser); err != nil {
		return errors.New("failed to update password")
	}

	// Clear reset token
	if err := s.userRepo.ClearPasswordResetToken(matchedUser.ID.Hex()); err != nil {
		// Log error but don't fail since password was updated
		return nil
	}

	return nil
}
