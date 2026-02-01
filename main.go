package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/nexus-im/nexus/store/conversation"
	"github.com/nexus-im/nexus/store/session"
	"github.com/nexus-im/nexus/store/user"

	"golang.org/x/crypto/bcrypt"

	_ "github.com/lib/pq"
)

var addr = flag.String("addr", ":8080", "http service address")

// Global instances (in a real app, use dependency injection)
var (
	userStore         user.Store
	sessionStore      session.Store
	conversationStore conversation.Store
)

const sessionTTL = 24 * time.Hour

func main() {
	flag.Parse()

	// Database Connection
	// TODO: Load from env
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://nexus_user:nexus_password@localhost:5432/nexus?sslmode=disable"
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Error closing db: %v", err)
		}
	}()

	if err := db.Ping(); err != nil {
		// Just log warning, maybe DB isn't up yet (Docker)
		log.Printf("Warning: Database unreachable: %v", err)
	} else {
		log.Println("Connected to database")
	}

	userStore = user.NewSQLStore(db)
	sessionStore = session.NewSQLStore(db)
	conversationStore = conversation.NewSQLStore(db)

	hub := newHub()
	go hub.run()

	// API Endpoints
	http.HandleFunc("/api/register", handleRegister)
	http.HandleFunc("/api/login", handleLogin)
	http.HandleFunc("/api/conversations", handleCreateConversation)

	// WebSocket Endpoint
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	// Health Check
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			log.Printf("health check write error: %v", err)
		}
	})

	log.Printf("Server starting on %s", *addr)
	err = http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// Hash Password
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create User
	// Note: ID generation should ideally happen in DB (UUID) or here if we use a library.
	// The DB schema says `default: gen_random_uuid()`, so we can pass empty ID or handle it.
	// Our User struct has ID string. If we pass empty string to Postgres UUID column it might fail if not handled.
	// Let's assume the store/DB handles ID generation if we don't provide one,
	// OR we generate one here. For simplicity let's see if the store handles it.
	// Looking at sql_store.go, it inserts the ID. So we need to generate it.
	// Ideally, we should let the DB generate it and use `RETURNING id`.
	// For now, I'll use a placeholder logic or rely on DB default if I modify the query.
	// Actually, let's just generate a simple random ID for now to keep it moving,
	// or modify the store to support RETURNING.

	// Modifying the store to support DB-generated IDs is better, but let's stick to the current store impl.
	// I'll assume we need to provide an ID.
	newUser := &user.User{
		Username:     req.Username,
		PasswordHash: string(hashedBytes),
		CreatedAt:    time.Now(),
		LastSeen:     time.Now(),
	}

	if err := userStore.Create(r.Context(), newUser); err != nil {
		if err == user.ErrDuplicateUsername {
			http.Error(w, "Username already exists", http.StatusConflict)
			return
		}
		log.Printf("Error creating user: %v", err)
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	u, err := userStore.GetByUsername(r.Context(), req.Username)
	if err != nil {
		if err == user.ErrUserNotFound {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := generateSessionToken()
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	now := time.Now()
	sess := &session.Session{
		UserID:    u.ID,
		Token:     token,
		CreatedAt: now,
		ExpiresAt: now.Add(sessionTTL),
	}

	if err := sessionStore.Create(r.Context(), sess); err != nil {
		log.Printf("Error creating session: %v", err)
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	resp := map[string]interface{}{
		"token":      token,
		"expires_in": int(sessionTTL.Seconds()),
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("login response write error: %v", err)
	}
}

func handleCreateConversation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, err := authenticateRequest(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Type      string   `json:"type"`
		UserID    string   `json:"user_id"`
		MemberIDs []string `json:"member_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	switch req.Type {
	case string(conversation.TypeP2P):
		if req.UserID == "" {
			http.Error(w, "user_id is required for p2p conversations", http.StatusBadRequest)
			return
		}
		var existing *conversation.Conversation
		if req.UserID == userID {
			existing, err = conversationStore.GetSelfP2P(r.Context(), userID)
		} else {
			existing, err = conversationStore.GetP2PBetween(r.Context(), userID, req.UserID)
		}
		if err == nil {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"conversation_id": existing.ID,
				"created":         false,
			})
			return
		}
		if err != conversation.ErrConversationNotFound {
			http.Error(w, "Failed to look up conversation", http.StatusInternalServerError)
			return
		}

		convo := &conversation.Conversation{
			Type:      conversation.TypeP2P,
			CreatedBy: userID,
			CreatedAt: time.Now(),
		}
		members := []string{userID}
		if req.UserID != userID {
			members = append(members, req.UserID)
		}
		if err := conversationStore.CreateConversation(r.Context(), convo, members); err != nil {
			log.Fatal("Failed to create conversation:", err)
			http.Error(w, "Failed to create conversation", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"conversation_id": convo.ID,
			"created":         true,
		})
		return

	case string(conversation.TypeGroup):
		if len(req.MemberIDs) == 0 {
			http.Error(w, "member_ids is required for group conversations", http.StatusBadRequest)
			return
		}

		memberSet := map[string]struct{}{userID: {}}
		for _, id := range req.MemberIDs {
			if id == "" {
				continue
			}
			memberSet[id] = struct{}{}
		}

		members := make([]string, 0, len(memberSet))
		for id := range memberSet {
			members = append(members, id)
		}

		convo := &conversation.Conversation{
			Type:      conversation.TypeGroup,
			CreatedBy: userID,
			CreatedAt: time.Now(),
		}
		if err := conversationStore.CreateConversation(r.Context(), convo, members); err != nil {
			http.Error(w, "Failed to create conversation", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"conversation_id": convo.ID,
			"created":         true,
		})
		return
	default:
		http.Error(w, "Invalid conversation type", http.StatusBadRequest)
		return
	}
}

func authenticateRequest(r *http.Request) (string, error) {
	token := strings.TrimSpace(r.Header.Get("X-Session-Token"))
	if token == "" {
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		}
	}
	if token == "" {
		return "", session.ErrSessionNotFound
	}

	sess, err := sessionStore.GetByToken(r.Context(), token)
	if err != nil {
		return "", err
	}
	return sess.UserID, nil
}

func generateSessionToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
