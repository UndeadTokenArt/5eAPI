package auth

import (
	"context"

	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Login(ctx context.Context, creds Credentials) (*TokenPair, error) {
	user, err := s.repo.GetUserByEmail(ctx, creds.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Compare stored password hash with the provided password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(creds.Password))
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Generate and return token pair using user.ID
	return &TokenPair{
		AccessToken:  "mock-access-token",
		RefreshToken: "mock-refresh-token",
	}, nil
}
