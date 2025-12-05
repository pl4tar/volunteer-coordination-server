package entity

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"user_id,omitempty" db:"id"`
	Email        string    `json:"user_email" db:"email"`
	PasswordHash string    `json:"password" db:"password_hash"`
	FullName     string    `json:"full_name" db:"full_name"`
	PhoneNumber  string    `json:"phone_number" db:"phone"`
	Role         string    `json:"role,omitempty" db:"role"`
	IsBlocked    bool      `json:"is_blocked,omitempty" db:"is_blocked"`
	CreatedAt    time.Time `json:"created_at,omitempty" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at,omitempty" db:"updated_at"`
}

type UserRegistration struct {
	Email       string    `json:"user_email" db:"email"`
	Password    string    `json:"password" db:"password_hash"`
	FullName    string    `json:"full_name" db:"full_name"`
	PhoneNumber string    `json:"phone_number" db:"phone"`
	Role        string    `json:"role" db:"role"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}
type Users []User

func (u Users) Len() int {
	return len(u)
}
