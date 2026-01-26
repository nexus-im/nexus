package user

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCreate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Logf("error closing db: %v", closeErr)
		}
	}()

	store := NewSQLStore(db)
	ctx := context.Background()

	fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	u := &User{
		ID:           "user-123",
		Username:     "testuser",
		PasswordHash: "hashedsecret",
		CreatedAt:    fixedTime,
		LastSeen:     fixedTime,
	}

	// Expectation
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO users (username, password_hash, created_at, last_seen) VALUES ($1, $2, $3, $4) RETURNING id`)).
		WithArgs(u.Username, u.PasswordHash, u.CreatedAt, u.LastSeen).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(u.ID))

	err = store.Create(ctx, u)
	if err != nil {
		t.Errorf("error was not expected while updating stats: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Logf("error closing db: %v", closeErr)
		}
	}()

	store := NewSQLStore(db)
	ctx := context.Background()

	userID := "user-123"
	fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	// Success Case
	rows := sqlmock.NewRows([]string{"id", "username", "password_hash", "created_at", "last_seen"}).
		AddRow(userID, "testuser", "hashedsecret", fixedTime, fixedTime)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, username, password_hash, created_at, last_seen FROM users WHERE id = $1`)).
		WithArgs(userID).
		WillReturnRows(rows)

	u, err := store.GetByID(ctx, userID)
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}
	if u == nil {
		t.Errorf("expected user, got nil")
	} else if u.ID != userID {
		t.Errorf("expected id %s, got %s", userID, u.ID)
	}

	// Not Found Case
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, username, password_hash, created_at, last_seen FROM users WHERE id = $1`)).
		WithArgs("unknown").
		WillReturnError(sql.ErrNoRows)

	_, err = store.GetByID(ctx, "unknown")
	if err != ErrUserNotFound {
		t.Errorf("expected error ErrUserNotFound, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetByUsername(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Logf("error closing db: %v", closeErr)
		}
	}()

	store := NewSQLStore(db)
	ctx := context.Background()

	username := "testuser"
	fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	// Success Case
	rows := sqlmock.NewRows([]string{"id", "username", "password_hash", "created_at", "last_seen"}).
		AddRow("user-123", username, "hashedsecret", fixedTime, fixedTime)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, username, password_hash, created_at, last_seen FROM users WHERE username = $1`)).
		WithArgs(username).
		WillReturnRows(rows)

	u, err := store.GetByUsername(ctx, username)
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}
	if u == nil {
		t.Errorf("expected user, got nil")
	} else if u.Username != username {
		t.Errorf("expected username %s, got %s", username, u.Username)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestUpdateLastSeen(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Logf("error closing db: %v", closeErr)
		}
	}()

	store := NewSQLStore(db)
	ctx := context.Background()

	userID := "user-123"
	newTime := time.Date(2023, 1, 2, 12, 0, 0, 0, time.UTC)

	// Success Case
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE users SET last_seen = $1 WHERE id = $2`)).
		WithArgs(newTime, userID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = store.UpdateLastSeen(ctx, userID, newTime)
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}

	// Not Found Case (0 rows affected)
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE users SET last_seen = $1 WHERE id = $2`)).
		WithArgs(newTime, "unknown").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = store.UpdateLastSeen(ctx, "unknown", newTime)
	if err != ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
