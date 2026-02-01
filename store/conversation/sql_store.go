package conversation

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

func (s *SQLStore) GetP2PBetween(ctx context.Context, userAID, userBID string) (*Conversation, error) {
	query := `
		SELECT c.id, c.type, c.created_by, c.created_at
		FROM conversations c
		JOIN conversation_members m1 ON m1.conversation_id = c.id
		JOIN conversation_members m2 ON m2.conversation_id = c.id
		WHERE c.type = 'p2p'
			AND m1.user_id = $1
			AND m2.user_id = $2
		LIMIT 1
	`

	row := s.db.QueryRowContext(ctx, query, userAID, userBID)

	var convo Conversation
	if err := row.Scan(&convo.ID, &convo.Type, &convo.CreatedBy, &convo.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrConversationNotFound
		}
		return nil, err
	}

	return &convo, nil
}

func (s *SQLStore) GetSelfP2P(ctx context.Context, userID string) (*Conversation, error) {
	query := `
		SELECT c.id, c.type, c.created_by, c.created_at
		FROM conversations c
		JOIN conversation_members m ON m.conversation_id = c.id
		WHERE c.type = 'p2p' AND m.user_id = $1
		GROUP BY c.id
		HAVING COUNT(*) = 1
		LIMIT 1
	`

	row := s.db.QueryRowContext(ctx, query, userID)

	var convo Conversation
	if err := row.Scan(&convo.ID, &convo.Type, &convo.CreatedBy, &convo.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrConversationNotFound
		}
		return nil, err
	}

	return &convo, nil
}

func (s *SQLStore) CreateConversation(ctx context.Context, convo *Conversation, memberIDs []string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				_ = rollbackErr
			}
		}
	}()

	if convo.CreatedAt.IsZero() {
		convo.CreatedAt = time.Now()
	}

	convoInsert := `
		INSERT INTO conversations (type, created_by, created_at)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	if err = tx.QueryRowContext(ctx, convoInsert, convo.Type, convo.CreatedBy, convo.CreatedAt).Scan(&convo.ID); err != nil {
		return err
	}

	memberInsert := `
		INSERT INTO conversation_members (conversation_id, user_id, joined_at)
		VALUES ($1, $2, $3)
	`

	joinedAt := time.Now()
	for _, memberID := range memberIDs {
		if _, err = tx.ExecContext(ctx, memberInsert, convo.ID, memberID, joinedAt); err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}
