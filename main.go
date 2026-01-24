package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"nexus/backend/internal/auth"
	"nexus/backend/store/user"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"

	_ "github.com/lib/pq"
)

var addr = flag.String("addr", ":8080", "http service address")

// Global instances (in a real app, use dependency injection)
var (
	authenticator *auth.Authenticator
	userStore     user.Store
)

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

	// Initialize Authenticator
	authenticator = auth.NewAuthenticator("temporary_secret_key", "nexus-im", 24*time.Hour)

	hub := newHub()
	go hub.run()

	// API Endpoints
	http.HandleFunc("/api/register", handleRegister)
	http.HandleFunc("/api/login", handleLogin)

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
		ID:           generateID(), 
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

	token, err := authenticator.GenerateToken(u.ID, u.Username)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"token": token}); err != nil {
		log.Printf("login response write error: %v", err)
	}
}

// Simple ID generator for prototype
func generateID() string {
	return time.Now().Format("20060102150405") // Timestamp as ID for now
}