package conversation

import (
	"context"
	"errors"
	"time"
)

type Type string

const (
	TypeP2P   Type = "p2p"
	TypeGroup Type = "group"
)

// Conversation represents a chat thread between users.
type Conversation struct {
	ID        string    `json:"id"`
	Type      Type      `json:"type"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

var (
	ErrConversationNotFound = errors.New("conversation not found")
)

// Store defines conversation persistence operations.
type Store interface {
	GetP2PBetween(ctx context.Context, userAID, userBID string) (*Conversation, error)
	GetSelfP2P(ctx context.Context, userID string) (*Conversation, error)
	CreateConversation(ctx context.Context, convo *Conversation, memberIDs []string) error
}
