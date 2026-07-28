package auth

import "context"

type Repository interface {
	CreateUser(ctx context.Context, user *User) error
	GetUserByEmail(ctx context.Context, email string) (*User, error)
}

// Example implementation struct (e.g., PostgresRepository)
