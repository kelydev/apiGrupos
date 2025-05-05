package models

import "time"

// Investigador represents an investigator in the database.
type Investigador struct {
	ID        int       `json:"idInvestigador" db:"idInvestigador"`
	Nombre    string    `json:"nombre" db:"nombre"`
	Apellido  string    `json:"apellido" db:"apellido"`
	CreatedAt time.Time `json:"createdAt" db:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" db:"updatedAt"`
}

// InvestigadorConRol represents an investigator with their specific role within a group.
type InvestigadorConRol struct {
	ID        int       `json:"idInvestigador"`
	Nombre    string    `json:"nombre"`
	Apellido  string    `json:"apellido"`
	Rol       string    `json:"rol"` // Role within the specific group
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
