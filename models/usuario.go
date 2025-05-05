package models

import "time"

// Usuario represents a user in the application database.
type Usuario struct {
	ID        int       `json:"idUsuario" db:"idusuario"` // Use lowercase db tag
	Email     string    `json:"email" db:"email"`
	Password  string    `json:"-" db:"password"` // Exclude password hash from JSON responses
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// Credentials represents the data needed for login.
type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
