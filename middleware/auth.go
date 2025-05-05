package middleware

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// Define a key type for context values to avoid collisions
type contextKey string

const (
	// UserIDKey is the key used to store the user ID in the request context
	UserIDKey contextKey = "userID"
)

// JWTMiddleware verifies the JWT token from the Authorization header.
func JWTMiddleware(next http.Handler) http.Handler {
	// Get the secret key from environment variable
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		// Log fatal error if secret is not set, as the app cannot securely function
		log.Fatal("FATAL: JWT_SECRET environment variable not set.")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Get the token from the Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		// Check if the header is in the format "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.Error(w, "Authorization header format must be Bearer {token}", http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]

		// 2. Parse and validate the token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Don't forget to validate the alg is what you expect:
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			// Return the secret key for validation
			return []byte(jwtSecret), nil
		})

		if err != nil {
			log.Printf("Token validation error: %v", err)
			// Check for specific JWT error types using errors.Is
			if errors.Is(err, jwt.ErrTokenMalformed) {
				http.Error(w, "Malformed token", http.StatusUnauthorized)
			} else if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
				http.Error(w, "Invalid token signature", http.StatusUnauthorized)
			} else if errors.Is(err, jwt.ErrTokenExpired) || errors.Is(err, jwt.ErrTokenNotValidYet) {
				http.Error(w, "Token is either expired or not active yet", http.StatusUnauthorized)
			} else {
				// Other errors (e.g., network issues during key fetch if using JWKS, or other validation errors)
				http.Error(w, "Couldn't handle this token: validation error", http.StatusUnauthorized)
			}
			return
		}

		if !token.Valid {
			// This case should ideally not be reached if the checks above are exhaustive
			// but kept as a fallback.
			http.Error(w, "Invalid token (general validation failed)", http.StatusUnauthorized)
			return
		}

		// 3. Optional: Extract claims (e.g., user ID) and add to context
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			// Example: Extract 'sub' (subject) claim, often used for user ID
			if userID, ok := claims["sub"].(string); ok { // Assuming user ID is a string in 'sub'
				// Add user ID to context
				ctx := context.WithValue(r.Context(), UserIDKey, userID)
				r = r.WithContext(ctx)
			} else {
				// Handle case where 'sub' claim is missing or not a string if it's mandatory
				// log.Printf("Warning: 'sub' claim missing or not a string in token")
			}
			// You can extract other claims similarly
		} else {
			log.Printf("Warning: Could not parse token claims")
		}

		// 4. Call the next handler if the token is valid
		next.ServeHTTP(w, r)
	})
}
