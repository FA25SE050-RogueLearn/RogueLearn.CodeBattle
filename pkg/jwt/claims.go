package jwt

import (
	jwt "github.com/golang-jwt/jwt/v5"
)

type UserClaims struct {
	ID       string   `json:"id"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}
