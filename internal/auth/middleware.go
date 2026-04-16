package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
)

// ContextKey is a custom type for context keys
type ContextKey string

const (
	// UserContextKey is the key used to store user info in context
	UserContextKey ContextKey = "user"
)

// UserInfo contains user information stored in context
type UserInfo struct {
	UserID   string
	Username string
	Email    string
	Role     string
	FullName string
}

const AuthCookieName = "ponches_auth"

func ExtractTokenFromRequest(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && parts[0] == "Bearer" {
			return parts[1]
		}
	}

	if cookie, err := r.Cookie(AuthCookieName); err == nil && cookie.Value != "" {
		return cookie.Value
	}

	return ""
}

// Middleware creates a JWT authentication middleware
func (s *JWTService) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := ExtractTokenFromRequest(r)

		if tokenString == "" {
			http.Error(w, `{"error": "Authorization required"}`, http.StatusUnauthorized)
			return
		}

		// Validate token
		claims, err := s.ValidateToken(tokenString)
		if err != nil {
			log.Warn().Err(err).Msg("Invalid JWT token")
			http.Error(w, `{"error": "Invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		// Add user info to context
		userInfo := &UserInfo{
			UserID:   claims.UserID,
			Username: claims.Username,
			Email:    claims.Email,
			Role:     claims.Role,
			FullName: claims.FullName,
		}

		ctx := context.WithValue(r.Context(), UserContextKey, userInfo)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserFromContext retrieves user info from context
func GetUserFromContext(ctx context.Context) (*UserInfo, bool) {
	user, ok := ctx.Value(UserContextKey).(*UserInfo)
	return user, ok
}

// RequireRole creates a middleware that requires a specific role
func RequireRole(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := GetUserFromContext(r.Context())
			if !ok {
				http.Error(w, `{"error": "User not found in context"}`, http.StatusInternalServerError)
				return
			}

			// Check if user role is in allowed roles
			allowed := false
			for _, role := range allowedRoles {
				if user.Role == role {
					allowed = true
					break
				}
			}

			if !allowed {
				http.Error(w, `{"error": "Insufficient permissions"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
