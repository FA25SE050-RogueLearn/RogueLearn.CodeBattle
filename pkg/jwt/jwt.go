// jwt package provides function to `read and parse` token
package jwt

import (
	"errors"
	"log/slog"

	"github.com/golang-jwt/jwt/v5"
)

type JWTParser struct {
	logger    *slog.Logger
	secretKey string
}

func NewJWTParser(secretKey string, logger *slog.Logger) *JWTParser {
	return &JWTParser{
		logger:    logger,
		secretKey: secretKey,
	}
}

func (p *JWTParser) VerifyToken(tokenStr string) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &UserClaims{}, func(token *jwt.Token) (any, error) {
		_, ok := token.Method.(*jwt.SigningMethodHMAC)

		if !ok {
			return nil, errors.New("error parsing token")
		}
		return []byte(p.secretKey), nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token claims")
}

func (p *JWTParser) GetUserClaimsFromToken(tokenStr string) (*UserClaims, error) {
	// Parse the token and extract claims
	token, err := jwt.ParseWithClaims(tokenStr, &UserClaims{}, func(token *jwt.Token) (any, error) {
		_, ok := token.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			p.logger.Error("Unexpected signing method", "method", token.Method)
			return nil, errors.New("unexpected signing method")
		}
		return []byte(p.secretKey), nil
	})

	if err != nil && !errors.Is(err, jwt.ErrTokenExpired) {
		p.logger.Error("Failed to parse jwt at 54", "err", err)
		return nil, err
	}

	if claims := token.Claims.(*UserClaims); claims != nil {
		return claims, nil
	}

	return nil, errors.New("invalid token claims")
}
