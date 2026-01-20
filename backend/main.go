package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"nexus/backend/internal/auth"
	"time"
)

var addr = flag.String("addr", ":8080", "http service address")

// Global authenticator for simplicity in this prototype.
// In a real app, inject this via a struct.
var authenticator *auth.Authenticator

func main() {
	flag.Parse()

	// Initialize Authenticator
	// TODO: Load secret from environment variable
	authenticator = auth.NewAuthenticator("temporary_secret_key", "nexus-im", 24*time.Hour)

	hub := newHub()
	go hub.run()

	// Login Endpoint
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
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	// In a real app, verify password here.
	// For this prototype, we just generate a token for the username.
	userID := "user_" + req.Username // Mock ID

	token, err := authenticator.GenerateToken(userID, req.Username)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"token": token}); err != nil {
		log.Printf("login response write error: %v", err)
	}
}
