package oidc

import (
	"context"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// Middleware returns an HTTP middleware that validates Bearer tokens.
// If the validator has no provider configured, requests pass through (lever not active).
// If the provider is configured and validation fails, returns 401.
func (v *Validator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			// No token — if OIDC is configured, reject; if not, pass through
			if err := v.Validate(r.Context(), ""); err == ErrNotConfigured {
				next.ServeHTTP(w, r)
				return
			}
			log.WithField("path", r.URL.Path).Debug("oidc: missing authorization header")
			http.Error(w, "authorization required", http.StatusUnauthorized)
			return
		}

		rawToken := strings.TrimPrefix(authHeader, "Bearer ")
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		if err := v.Validate(ctx, rawToken); err != nil {
			if err == ErrNotConfigured {
				next.ServeHTTP(w, r)
				return
			}
			log.WithField("path", r.URL.Path).WithError(err).Debug("oidc: token validation failed")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
