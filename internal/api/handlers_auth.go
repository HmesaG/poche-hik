package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"ponches/internal/auth"
	"ponches/internal/users"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func isSecureRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func setAuthCookie(w http.ResponseWriter, r *http.Request, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.AuthCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   isSecureRequest(r),
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
	})
}

func clearAuthCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.AuthCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   isSecureRequest(r),
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}

// handleLogin handles user login
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var loginReq users.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if loginReq.Username == "" || loginReq.Password == "" {
		writeError(w, http.StatusBadRequest, "Username and password are required")
		return
	}

	// Get user from database
	user, err := s.Store.GetUserByUsername(r.Context(), loginReq.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to authenticate")
		return
	}

	if user == nil || !user.Active {
		writeError(w, http.StatusUnauthorized, "Invalid credentials or inactive account")
		return
	}

	// Verify password
	if !auth.CheckPasswordHash(loginReq.Password, user.Password) {
		writeError(w, http.StatusUnauthorized, "Invalid credentials or inactive account")
		return
	}

	// Generate JWT token
	token, expiresAt, err := s.JWTService.GenerateToken(user.ID, user.Username, user.Email, user.Role, user.FullName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}
	setAuthCookie(w, r, token, expiresAt)

	response := users.LoginResponse{
		Token:     "",
		ExpiresAt: expiresAt.Unix(),
		User: users.User{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			FullName: user.FullName,
			Role:     user.Role,
		},
	}

	writeJSON(w, http.StatusOK, response)
}

// handleLogout handles user logout (client should discard token)
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	clearAuthCookie(w, r)
	writeJSON(w, http.StatusOK, map[string]string{"message": "Logged out successfully"})
}

// handleMe returns current user info
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":       user.UserID,
		"username": user.Username,
		"email":    user.Email,
		"role":     user.Role,
		"fullName": user.FullName,
	})
}

// handleRegisterUser handles user registration (admin only)
func (s *Server) handleRegisterUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
		FullName string `json:"fullName"`
		Role     string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if req.Username == "" || req.Password == "" || req.Email == "" || req.FullName == "" {
		writeError(w, http.StatusBadRequest, "Username, password, email, and fullName are required")
		return
	}
	if len(req.Password) < 6 {
		writeError(w, http.StatusBadRequest, "Password must be at least 6 characters")
		return
	}

	// Validate role
	validRoles := map[string]bool{"admin": true, "manager": true, "viewer": true}
	if req.Role == "" {
		req.Role = "viewer"
	}
	if !validRoles[req.Role] {
		writeError(w, http.StatusBadRequest, "Invalid role. Valid values: admin, manager, viewer")
		return
	}

	// Check if username already exists
	existingUser, err := s.Store.GetUserByUsername(r.Context(), req.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to validate username")
		return
	}
	if existingUser != nil {
		writeError(w, http.StatusConflict, "Username already exists")
		return
	}

	// Hash password
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to process password")
		return
	}

	// Create user
	user := &users.User{
		ID:       uuid.New().String(),
		Username: req.Username,
		Email:    req.Email,
		Password: hashedPassword,
		FullName: req.FullName,
		Role:     req.Role,
		Active:   true,
	}

	if err := s.Store.CreateUser(r.Context(), user); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			writeError(w, http.StatusConflict, "Username or email already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	// Return user without password
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"fullName": user.FullName,
		"role":     user.Role,
		"active":   user.Active,
	})
}

// handleListUsers returns all users (admin only)
func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.Store.ListUsers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list users")
		return
	}

	// Remove passwords from response
	response := make([]map[string]interface{}, len(users))
	for i, u := range users {
		response[i] = map[string]interface{}{
			"id":        u.ID,
			"username":  u.Username,
			"email":     u.Email,
			"fullName":  u.FullName,
			"role":      u.Role,
			"active":    u.Active,
			"createdAt": u.CreatedAt,
			"updatedAt": u.UpdatedAt,
		}
	}

	writeJSON(w, http.StatusOK, response)
}

// handleDeleteUser deletes a user (admin only)
func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "User ID is required")
		return
	}

	if err := s.Store.DeleteUser(r.Context(), id); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "User not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to delete user")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleUpdateUser updates a user (admin only)
func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "User ID is required")
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
		FullName string `json:"fullName"`
		Role     string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get existing user
	user, err := s.Store.GetUser(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}
	if user == nil {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}

	// Update fields if provided
	if req.Username != "" {
		// Check if username is taken by another user
		existing, err := s.Store.GetUserByUsername(r.Context(), req.Username)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to validate username")
			return
		}
		if existing != nil && existing.ID != id {
			writeError(w, http.StatusConflict, "Username already exists")
			return
		}
		user.Username = req.Username
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.FullName != "" {
		user.FullName = req.FullName
	}
	if req.Role != "" {
		validRoles := map[string]bool{"admin": true, "manager": true, "viewer": true}
		if !validRoles[req.Role] {
			writeError(w, http.StatusBadRequest, "Invalid role")
			return
		}
		user.Role = req.Role
	}
	if req.Password != "" {
		if len(req.Password) < 6 {
			writeError(w, http.StatusBadRequest, "Password must be at least 6 characters")
			return
		}
		hashedPassword, err := auth.HashPassword(req.Password)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to process password")
			return
		}
		user.Password = hashedPassword
	}

	if err := s.Store.UpdateUser(r.Context(), user); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "User not found")
			return
		}
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			writeError(w, http.StatusConflict, "Username or email already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to update user")
		return
	}

	// Return user without password
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"fullName": user.FullName,
		"role":     user.Role,
		"active":   user.Active,
	})
}

// handleGetUser returns a single user by ID
func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "User ID is required")
		return
	}

	user, err := s.Store.GetUser(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}
	if user == nil {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}

	// Return user without password
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"fullName": user.FullName,
		"role":     user.Role,
		"active":   user.Active,
	})
}
