package api

import (
	"context"
	"net/http"
	"strings"
)

type Key string

const (
	UserClaimsKey Key = "user_claims"
)

func (app *Application) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			app.logger.Warn("Missing Authorization header")
			app.unauthorized(w, r)
			return
		}

		headerParts := strings.Split(authHeader, " ")
		if len(headerParts) != 2 || strings.ToLower(headerParts[0]) != "bearer" {
			app.logger.Warn("Malformed Authorization header", "header", authHeader)
			app.unauthorized(w, r)
			return
		}

		tokenStr := headerParts[1]
		// verify token here (with jwt secret key)
		claims, err := app.jwtParser.GetUserClaimsFromToken(tokenStr)
		if err != nil {
			app.logger.Error("Failed to parse token", "error", err)
			app.unauthorized(w, r)
			return
		}

		app.logger.Info("Auth endpoint hit", "claims", claims)

		ctx := context.WithValue(r.Context(), UserClaimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
