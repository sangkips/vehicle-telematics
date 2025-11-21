package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID                  primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username            string             `bson:"username" json:"username" validate:"required,min=3,max=50"`
	Email               string             `bson:"email" json:"email" validate:"required,email"`
	FirstName           string             `bson:"first_name" json:"firstName" validate:"required"`
	LastName            string             `bson:"last_name" json:"lastName" validate:"required"`
	Password            string             `bson:"password" json:"-"`
	Role                string             `bson:"role" json:"role" validate:"required,oneof=admin manager operator viewer"`
	Status              string             `bson:"status" json:"status" validate:"required,oneof=active inactive suspended"`
	Permissions         []string           `bson:"permissions" json:"permissions"`
	PasswordResetToken  string             `bson:"password_reset_token,omitempty" json:"-"`
	PasswordResetExpiry *time.Time         `bson:"password_reset_expiry,omitempty" json:"-"`
	LastLogin           *time.Time         `bson:"last_login,omitempty" json:"lastLogin,omitempty"`
	CreatedAt           time.Time          `bson:"created_at" json:"createdAt"`
	UpdatedAt           time.Time          `bson:"updated_at" json:"updatedAt"`
}

type AuthUser struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	FirstName   string   `json:"firstName"`
	LastName    string   `json:"lastName"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
}
