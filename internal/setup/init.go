package setup

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"ponches/internal/auth"
	"ponches/internal/store"
	"ponches/internal/users"

	"github.com/google/uuid"
)

// InitDefaultAdmin creates a bootstrap admin user if none exists.
func InitDefaultAdmin(s store.Repository) error {
	ctx := context.Background()

	admin, _ := s.GetUserByUsername(ctx, "admin")
	if admin != nil {
		return nil
	}

	password := os.Getenv("DEFAULT_ADMIN_PASSWORD")
	if password == "" {
		var err error
		password, err = generateBootstrapPassword()
		if err != nil {
			return fmt.Errorf("failed to generate bootstrap admin password: %w", err)
		}
	}

	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	adminUser := &users.User{
		ID:       uuid.New().String(),
		Username: "admin",
		Email:    "admin@ponches.local",
		Password: hashedPassword,
		FullName: "Administrador",
		Role:     "admin",
		Active:   true,
	}

	if err := s.CreateUser(ctx, adminUser); err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	fmt.Printf("Bootstrap admin user created (username: admin, password: %s)\n", password)
	return nil
}

func generateBootstrapPassword() (string, error) {
	bytes := make([]byte, 18)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}
