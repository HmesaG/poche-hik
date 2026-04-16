package store

import (
	"context"
	"database/sql"
	"ponches/internal/users"
	"time"
)

// UserStore defines the interface for user data persistence
type UserStore interface {
	CreateUser(ctx context.Context, u *users.User) error
	GetUser(ctx context.Context, id string) (*users.User, error)
	GetUserByUsername(ctx context.Context, username string) (*users.User, error)
	ListUsers(ctx context.Context) ([]*users.User, error)
	UpdateUser(ctx context.Context, u *users.User) error
	DeleteUser(ctx context.Context, id string) error
}

// CreateUser creates a new user
func (s *SQLiteStore) CreateUser(ctx context.Context, u *users.User) error {
	now := time.Now()
	u.CreatedAt = now
	u.UpdatedAt = now

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO users (id, username, email, password, full_name, role, active, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		u.ID, u.Username, u.Email, u.Password, u.FullName, u.Role, u.Active, now, now)
	return err
}

// GetUserByID retrieves a user by ID
func (s *SQLiteStore) GetUserByID(ctx context.Context, id string) (*users.User, error) {
	u := &users.User{}
	var createdAt, updatedAt time.Time

	err := s.db.QueryRowContext(ctx,
		`SELECT id, username, email, password, full_name, role, active, created_at, updated_at
		 FROM users WHERE id = ?`, id).
		Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.FullName, &u.Role, &u.Active,
			&createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	u.CreatedAt = createdAt
	u.UpdatedAt = updatedAt
	return u, nil
}

// GetUser retrieves a user by ID (alias for GetUserByID)
func (s *SQLiteStore) GetUser(ctx context.Context, id string) (*users.User, error) {
	return s.GetUserByID(ctx, id)
}

// GetUserByUsername retrieves a user by username
func (s *SQLiteStore) GetUserByUsername(ctx context.Context, username string) (*users.User, error) {
	u := &users.User{}
	var createdAt, updatedAt time.Time

	err := s.db.QueryRowContext(ctx,
		`SELECT id, username, email, password, full_name, role, active, created_at, updated_at
		 FROM users WHERE username = ?`, username).
		Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.FullName, &u.Role, &u.Active,
			&createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	u.CreatedAt = createdAt
	u.UpdatedAt = updatedAt
	return u, nil
}

// ListUsers retrieves all users
func (s *SQLiteStore) ListUsers(ctx context.Context) ([]*users.User, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, username, email, password, full_name, role, active, created_at, updated_at
		 FROM users ORDER BY username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*users.User
	for rows.Next() {
		u := &users.User{}
		var createdAt, updatedAt time.Time

		err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.FullName, &u.Role,
			&u.Active, &createdAt, &updatedAt)
		if err != nil {
			return nil, err
		}
		u.CreatedAt = createdAt
		u.UpdatedAt = updatedAt
		list = append(list, u)
	}
	return list, nil
}

// UpdateUser updates an existing user
func (s *SQLiteStore) UpdateUser(ctx context.Context, u *users.User) error {
	u.UpdatedAt = time.Now()

	result, err := s.db.ExecContext(ctx,
		`UPDATE users SET username = ?, email = ?, password = ?, full_name = ?, role = ?,
		 active = ?, updated_at = ? WHERE id = ?`,
		u.Username, u.Email, u.Password, u.FullName, u.Role, u.Active, u.UpdatedAt, u.ID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// DeleteUser deletes a user
func (s *SQLiteStore) DeleteUser(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}
