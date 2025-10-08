package handlers

import (
	"context"
	"net/http"
	"strings"
)

type Key string

const (
	UserClaimsKey Key = "user_claims"
)

func (hr *HandlerRepo) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			hr.logger.Warn("Missing Authorization header")
			hr.unauthorized(w, r)
			return
		}

		headerParts := strings.Split(authHeader, " ")
		if len(headerParts) != 2 || strings.ToLower(headerParts[0]) != "bearer" {
			hr.logger.Warn("Malformed Authorization header", "header", authHeader)
			hr.unauthorized(w, r)
			return
		}

		tokenStr := headerParts[1]
		// verify token here (with jwt secret key)
		claims, err := hr.jwtParser.GetUserClaimsFromToken(tokenStr)
		if err != nil {
			hr.logger.Error("Failed to parse token", "error", err)
			hr.unauthorized(w, r)
			return
		}

		hr.logger.Info("Auth endpoint hit", "claims", claims)

		ctx := context.WithValue(r.Context(), UserClaimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
