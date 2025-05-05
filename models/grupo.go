package models

import "time"

// Grupo represents a research group in the database.
type Grupo struct {
	ID                 int       `json:"idGrupo" db:"idGrupo"`
	Nombre             string    `json:"nombre" db:"nombre"`
	NumeroResolucion   string    `json:"numeroResolucion" db:"numeroResolucion"`
	LineaInvestigacion string    `json:"lineaInvestigacion" db:"lineaInvestigacion"`
	TipoInvestigacion  string    `json:"tipoInvestigacion" db:"tipoInvestigacion"`
	FechaRegistro      time.Time `json:"fechaRegistro" db:"fechaRegistro"`
	Archivo            *string   `json:"archivo" db:"archivo"`
	CreatedAt          time.Time `json:"createdAt" db:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt" db:"updatedAt"`
}

// GrupoWithInvestigadores represents a group with its associated investigators including their roles.
type GrupoWithInvestigadores struct {
	Grupo          Grupo                `json:"grupo"`
	Investigadores []InvestigadorConRol `json:"investigadores"`
}
