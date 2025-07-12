package services

import (
	"errors"
	"fleet-backend/internal/models"
	"fleet-backend/internal/repository"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	userRepo *repository.UserRepository
}

func NewUserService(userRepo *repository.UserRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
}

type CreateUserRequest struct {
	Username  string `json:"username" validate:"required,min=3,max=50"`
	Email     string `json:"email" validate:"required,email"`
	FirstName string `json:"firstName" validate:"required,min=1,max=50"`
	LastName  string `json:"lastName" validate:"required,min=1,max=50"`
	Password  string `json:"password" validate:"required,min=6"`
	Role      string `json:"role" validate:"required,oneof=admin manager operator viewer"`
}

type UpdateUserRequest struct {
	Email     string `json:"email,omitempty" validate:"omitempty,email"`
	FirstName string `json:"firstName,omitempty" validate:"omitempty,min=1,max=50"`
	LastName  string `json:"lastName,omitempty" validate:"omitempty,min=1,max=50"`
	Role      string `json:"role,omitempty" validate:"omitempty,oneof=admin manager operator viewer"`
	Status    string `json:"status,omitempty" validate:"omitempty,oneof=active inactive suspended"`
}

func (s *UserService) GetAllUsers() ([]*models.User, error) {
	return s.userRepo.FindAll()
}

func (s *UserService) GetUserByID(id string) (*models.User, error) {
	return s.userRepo.FindByID(id)
}

func (s *UserService) CreateUser(req *CreateUserRequest) (*models.User, error) {
	// Check if username already exists
	existingUser, _ := s.userRepo.FindByUsername(req.Username)
	if existingUser != nil {
		return nil, errors.New("username already exists")
	}

	// Check if email already exists
	existingUser, _ = s.userRepo.FindByEmail(req.Email)
	if existingUser != nil {
		return nil, errors.New("email already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}

	// Create user model
	user := &models.User{
		ID:          primitive.NewObjectID(),
		Username:    req.Username,
		Email:       req.Email,
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		Password:    string(hashedPassword),
		Role:        req.Role,
		Status:      "active",
		Permissions: s.getRolePermissions(req.Role),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return s.userRepo.Create(user)
}

func (s *UserService) UpdateUser(id string, req *UpdateUserRequest) (*models.User, error) {
	// Find existing user
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Update fields if provided
	if req.Email != "" {
		// Check if email is already taken by another user
		existingUser, _ := s.userRepo.FindByEmail(req.Email)
		if existingUser != nil && existingUser.ID.Hex() != id {
			return nil, errors.New("email already exists")
		}
		user.Email = req.Email
	}

	if req.FirstName != "" {
		user.FirstName = req.FirstName
	}

	if req.LastName != "" {
		user.LastName = req.LastName
	}

	if req.Role != "" {
		user.Role = req.Role
		user.Permissions = s.getRolePermissions(req.Role)
	}

	if req.Status != "" {
		user.Status = req.Status
	}

	user.UpdatedAt = time.Now()

	return s.userRepo.Update(id, user)
}

func (s *UserService) DeleteUser(id string) error {
	// Check if user exists
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return errors.New("user not found")
	}

	// Prevent deletion of admin users (optional business rule)
	if user.Role == "admin" {
		return errors.New("cannot delete admin users")
	}

	return s.userRepo.Delete(id)
}

func (s *UserService) ChangeUserStatus(id string, status string) (*models.User, error) {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("user not found")
	}

	user.Status = status
	user.UpdatedAt = time.Now()

	return s.userRepo.Update(id, user)
}

func (s *UserService) ChangePassword(id string, currentPassword, newPassword string) error {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return errors.New("user not found")
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(currentPassword)); err != nil {
		return errors.New("current password is incorrect")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("failed to hash new password")
	}

	user.Password = string(hashedPassword)
	user.UpdatedAt = time.Now()

	_, err = s.userRepo.Update(id, user)
	return err
}

func (s *UserService) getRolePermissions(role string) []string {
	switch role {
	case "admin":
		return []string{"all"}
	case "manager":
		return []string{
			"view_vehicles", "create_vehicles", "update_vehicles", "delete_vehicles",
			"view_users", "create_users", "update_users",
			"view_alerts", "resolve_alerts",
			"view_reports", "export_reports",
			"view_system_settings",
		}
	case "operator":
		return []string{
			"view_vehicles", "update_vehicles",
			"view_alerts", "resolve_alerts",
			"view_reports",
		}
	case "viewer":
		return []string{
			"view_vehicles", "view_alerts", "view_reports",
		}
	default:
		return []string{}
	}
}