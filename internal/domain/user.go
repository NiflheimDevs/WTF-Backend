package domain

import (
	"time"

	"github.com/google/uuid"
)

// Role represents user role in the system
type Role string

const (
	RoleDispatcher Role = "dispatcher"
	RoleAdmin      Role = "admin"
)

// User represents a system user (dispatcher or admin)
type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Never expose in JSON
	FullName     string    `json:"full_name"`
	Role         Role      `json:"role"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// IsDispatcher checks if user has dispatcher role
func (u *User) IsDispatcher() bool {
	return u.Role == RoleDispatcher || u.Role == RoleAdmin
}

// IsAdmin checks if user has admin role
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}
