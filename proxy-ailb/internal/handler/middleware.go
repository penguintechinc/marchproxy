// Package handler provides HTTP request handlers and middleware for the AILB service.
package handler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/auth"
	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/metrics"
)

// LoggingMiddleware logs each request and its duration.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)

		slog.Info("request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.statusCode,
			"duration_ms", time.Since(start).Milliseconds(),
			"remote_addr", r.RemoteAddr,
		)
	})
}

// AuthMiddleware validates JWT tokens in the Authorization header.
// If the JWT secret is empty, all requests are allowed (development mode).
func AuthMiddleware(validator *auth.Validator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for health and metrics endpoints.
			if r.URL.Path == "/healthz" || r.URL.Path == "/metrics" {
				next.ServeHTTP(w, r)
				return
			}

			claims, err := validator.ValidateRequest(r)
			if err != nil {
				if err == auth.ErrNoAuthHeader {
					next.ServeHTTP(w, r)
					return
				}

				slog.Warn("auth failed", "error", err, "path", r.URL.Path)
				http.Error(w, `{"error":"unauthorized","message":"`+err.Error()+`"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// MetricsMiddleware tracks active request count via Prometheus gauge.
func MetricsMiddleware(m *metrics.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			m.ActiveRequests.Inc()
			defer m.ActiveRequests.Dec()
			next.ServeHTTP(w, r)
		})
	}
}

// RecoveryMiddleware catches panics and returns a 500 error instead of crashing.
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered", "error", rec, "path", r.URL.Path)
				http.Error(w, `{"error":"internal_server_error"}`, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

type contextKey string

const claimsKey contextKey = "claims"

// ClaimsFromContext extracts JWT claims from the request context.
func ClaimsFromContext(ctx context.Context) *auth.Claims {
	claims, _ := ctx.Value(claimsKey).(*auth.Claims)
	return claims
}
