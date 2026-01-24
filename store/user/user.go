package user

import (
	"context"
	"errors"
	"time"
)

// User represents a user account in the system.
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"` // Never export password hash to JSON
	CreatedAt    time.Time `json:"created_at"`
	LastSeen     time.Time `json:"last_seen"`
}

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrDuplicateUsername = errors.New("username already exists")
)

// Store defines the interface for CRUD operations on User accounts.
type Store interface {
	// CreateUser inserts a new user into the store.
	Create(ctx context.Context, user *User) error

	// GetByID retrieves a user by their unique ID.
	GetByID(ctx context.Context, id string) (*User, error)

	// GetByUsername retrieves a user by their username.
	GetByUsername(ctx context.Context, username string) (*User, error)

	// UpdateLastSeen updates the LastSeen timestamp for a user.
	UpdateLastSeen(ctx context.Context, id string, lastSeen time.Time) error
}
