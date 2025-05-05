package repository

import (
	"database/sql"
	"fmt"

	"github.com/GoogleCloudPlatform/golang-samples/run/helloworld/models"
	"golang.org/x/crypto/bcrypt"
)

// CreateUsuario inserts a new user into the database after hashing the password.
func CreateUsuario(db *sql.DB, u *models.Usuario) error {
	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("error hashing password: %w", err)
	}

	// Store the hashed password
	query := `INSERT INTO usuario (email, password) VALUES ($1, $2) RETURNING idusuario, created_at, updated_at`
	err = db.QueryRow(query, u.Email, string(hashedPassword)).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		// Consider checking for unique constraint violation on email
		return fmt.Errorf("error inserting user: %w", err)
	}

	// Clear the plaintext password from the struct after successful insertion
	u.Password = ""
	return nil
}

// GetUsuarioByEmail retrieves a user by their email address.
func GetUsuarioByEmail(db *sql.DB, email string) (*models.Usuario, error) {
	var u models.Usuario
	// Select all necessary fields, including the password hash
	query := `SELECT idusuario, email, password, created_at, updated_at FROM usuario WHERE email = $1`
	err := db.QueryRow(query, email).Scan(&u.ID, &u.Email, &u.Password, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // User not found, return nil error and nil user
		}
		return nil, fmt.Errorf("error getting user by email: %w", err)
	}
	return &u, nil
}

// CheckPasswordHash compares a plaintext password with a stored hash.
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil // Returns true if password matches hash
}
