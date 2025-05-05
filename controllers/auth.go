package controllers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/GoogleCloudPlatform/golang-samples/run/helloworld/models"
	"github.com/GoogleCloudPlatform/golang-samples/run/helloworld/repository"
	"github.com/golang-jwt/jwt/v5"
)

// RegisterHandler handles user registration.
func RegisterHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var creds models.Credentials // Use Credentials struct for input
		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Basic validation
		if creds.Email == "" || creds.Password == "" {
			http.Error(w, "Email and password are required", http.StatusBadRequest)
			return
		}
		// Add more validation if needed (e.g., password complexity, email format)

		// Check if user already exists
		existingUser, err := repository.GetUsuarioByEmail(db, creds.Email)
		if err != nil {
			log.Printf("Error checking for existing user: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if existingUser != nil {
			http.Error(w, "User with this email already exists", http.StatusConflict) // 409 Conflict
			return
		}

		// Create user model
		user := &models.Usuario{
			Email:    creds.Email,
			Password: creds.Password, // Pass plaintext password to repository
		}

		// Create user in repository (handles hashing)
		if err := repository.CreateUsuario(db, user); err != nil {
			log.Printf("Error creating user: %v", err)
			http.Error(w, "Failed to register user", http.StatusInternalServerError)
			return
		}

		// Respond with created user (password hash is excluded by JSON tag in model)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(user)
	}
}

// LoginHandler handles user login and JWT generation.
func LoginHandler(db *sql.DB) http.HandlerFunc {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("FATAL: JWT_SECRET environment variable not set for login handler.")
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var creds models.Credentials
		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if creds.Email == "" || creds.Password == "" {
			http.Error(w, "Email and password are required", http.StatusBadRequest)
			return
		}

		// Get user by email
		user, err := repository.GetUsuarioByEmail(db, creds.Email)
		if err != nil {
			log.Printf("Error fetching user for login: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if user == nil {
			// User not found
			http.Error(w, "Invalid email or password", http.StatusUnauthorized)
			return
		}

		// Compare the provided password with the stored hash
		if !repository.CheckPasswordHash(creds.Password, user.Password) {
			// Password doesn't match
			http.Error(w, "Invalid email or password", http.StatusUnauthorized)
			return
		}

		// --- Generate JWT Token ---
		// Set token claims
		expirationTime := time.Now().Add(24 * time.Hour) // Token valid for 24 hours
		claims := &jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   strconv.Itoa(user.ID), // Use user ID as subject
			// Issuer:    "your-app-name", // Optional: Add issuer
		}

		// Create token with claims
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

		// Generate encoded token and send it as response.
		tokenString, err := token.SignedString([]byte(jwtSecret))
		if err != nil {
			log.Printf("Error signing token: %v", err)
			http.Error(w, "Internal server error generating token", http.StatusInternalServerError)
			return
		}

		// --- Respond with the token ---
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"token": tokenString,
		})
	}
}
