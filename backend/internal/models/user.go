package models

import "time"

type UserRole string

const (
	RoleUser  UserRole = "user"
	RoleAdmin UserRole = "admin"
)

// from supabase auth
type User struct {
	ID           string    `json:"id"` // uuid
	Email        string    `json:"email"`
	CreatedAt    time.Time `json:"created_at"`
	LastSignInAt time.Time `json:"last_sign_in_at"`
	Role         UserRole  `json:"role"`
}
