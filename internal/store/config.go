package store

import (
	"context"
	"database/sql"
)

// ConfigStore defines the interface for configuration data persistence
type ConfigStore interface {
	GetConfigValue(ctx context.Context, key string) (string, error)
	SetConfigValue(ctx context.Context, key, value string) error
	GetAllConfig(ctx context.Context) (map[string]string, error)
}

// GetConfigValue retrieves a configuration value by key
func (s *SQLiteStore) GetConfigValue(ctx context.Context, key string) (string, error) {
	var value string
	err := s.db.QueryRowContext(ctx,
		`SELECT value FROM app_config WHERE key = ?`, key).
		Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil // Return empty string if key doesn't exist
	}
	return value, err
}

// SetConfigValue sets a configuration value (upsert)
func (s *SQLiteStore) SetConfigValue(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO app_config (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value)
	return err
}

// GetAllConfig retrieves all configuration values as a map
func (s *SQLiteStore) GetAllConfig(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT key, value FROM app_config ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	config := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		config[key] = value
	}
	return config, nil
}

// SetMultipleConfigValues sets multiple configuration values at once
func (s *SQLiteStore) SetMultipleConfigValues(ctx context.Context, values map[string]string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO app_config (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for key, value := range values {
		if _, err := stmt.ExecContext(ctx, key, value); err != nil {
			return err
		}
	}

	return tx.Commit()
}
