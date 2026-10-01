package auth

import "time"

type User struct {
	ID           string        `json:"id"`
	Email        string        `json:"email"`
	PasswordHash string        `json:"-"`
	CreatedAt    time.Time     `json:"created_at"`
	Inventory    UserInventory `json:"inventory"`
}

type UserInventory struct {
	Currency int     `json:"gold"`
	Items    []Items `json:"items"`
}

type Items struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type Credentials struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}
