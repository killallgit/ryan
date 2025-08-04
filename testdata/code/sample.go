package main

import (
	"context"
	"fmt"
	"time"
)

// UserService handles user-related operations
type UserService struct {
	db Database
}

// Database interface for data persistence
type Database interface {
	GetUser(ctx context.Context, id string) (*User, error)
	SaveUser(ctx context.Context, user *User) error
}

// User represents a user in the system
type User struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	Created  time.Time `json:"created"`
}

// NewUserService creates a new user service
func NewUserService(db Database) *UserService {
	return &UserService{db: db}
}

// CreateUser creates a new user with validation
func (us *UserService) CreateUser(ctx context.Context, name, email string) (*User, error) {
	if name == "" {
		return nil, fmt.Errorf("name cannot be empty")
	}
	
	if email == "" {
		return nil, fmt.Errorf("email cannot be empty")
	}

	user := &User{
		ID:      generateID(),
		Name:    name,
		Email:   email,
		Created: time.Now(),
	}

	err := us.db.SaveUser(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	return user, nil
}

// GetUser retrieves a user by ID
func (us *UserService) GetUser(ctx context.Context, id string) (*User, error) {
	if id == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	user, err := us.db.GetUser(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// generateID generates a unique identifier
func generateID() string {
	return fmt.Sprintf("user_%d", time.Now().UnixNano())
}

func main() {
	fmt.Println("User service example")
}