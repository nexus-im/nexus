package user

import (
	"context"
	"database/sql"
	"time"
)

// SQLStore implements Store using a database/sql connection.
type SQLStore struct {
	db *sql.DB
}

// NewSQLStore creates a new SQLStore.
func NewSQLStore(db *sql.DB) *SQLStore {
	return &SQLStore{db: db}
}

func (s *SQLStore) Create(ctx context.Context, user *User) error {
	query := `
		INSERT INTO users (id, username, password_hash, created_at, last_seen)
		VALUES ($1, $2, $3, $4, $5)
	`
	// Handle databases that might use ? instead of $1 (like SQLite) by default?
	// Or we just assume Postgres syntax for now as per design doc example?
	// To be safe for the design doc which showed Postgres, I will stick to Postgres placeholders ($1, $2).
	// If SQLite is used later, we might need to adjust or use a driver that supports $ placeholders.

	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now()
	}

	_, err := s.db.ExecContext(ctx, query,
		user.ID,
		user.Username,
		user.PasswordHash,
		user.CreatedAt,
		user.LastSeen,
	)

	if err != nil {
		// In a real app, we'd check for unique constraint violation here
		return err
	}

	return nil
}

func (s *SQLStore) GetByID(ctx context.Context, id string) (*User, error) {
	query := `SELECT id, username, password_hash, created_at, last_seen FROM users WHERE id = $1`

	row := s.db.QueryRowContext(ctx, query, id)

	var user User
	var lastSeen sql.NullTime // Handle nullable LastSeen

	err := row.Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.CreatedAt,
		&lastSeen,
	)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	} else if err != nil {
		return nil, err
	}

	if lastSeen.Valid {
		user.LastSeen = lastSeen.Time
	}

	return &user, nil
}

func (s *SQLStore) GetByUsername(ctx context.Context, username string) (*User, error) {
	query := `SELECT id, username, password_hash, created_at, last_seen FROM users WHERE username = $1`

	row := s.db.QueryRowContext(ctx, query, username)

	var user User
	var lastSeen sql.NullTime

	err := row.Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.CreatedAt,
		&lastSeen,
	)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	} else if err != nil {
		return nil, err
	}

	if lastSeen.Valid {
		user.LastSeen = lastSeen.Time
	}

	return &user, nil
}

func (s *SQLStore) UpdateLastSeen(ctx context.Context, id string, lastSeen time.Time) error {
	query := `UPDATE users SET last_seen = $1 WHERE id = $2`

	result, err := s.db.ExecContext(ctx, query, lastSeen, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}
