package repository

import (
	"database/sql"
	"fmt"

	"github.com/GoogleCloudPlatform/golang-samples/run/helloworld/models"
)

// CreateDetalleGrupoInvestigador inserts a new relationship between a group and an investigator.
func CreateDetalleGrupoInvestigador(db *sql.DB, detalle *models.DetalleGrupoInvestigador) error {
	// Usar nombres exactos de tabla y campos según la base de datos
	query := `INSERT INTO Grupo_Investigador (idGrupo, idInvestigador, rol) VALUES ($1, $2, $3) RETURNING idGrupo_Investigador, createdAt, updatedAt`
	err := db.QueryRow(query, detalle.IDGrupo, detalle.IDInvestigador, detalle.Rol).Scan(&detalle.ID, &detalle.CreatedAt, &detalle.UpdatedAt)
	if err != nil {
		return fmt.Errorf("error inserting group-investigator detail: %w", err)
	}
	return nil
}

// GetDetallesByGrupoID retrieves all relationship details for a given group ID.
func GetDetallesByGrupoID(db *sql.DB, grupoID int) ([]models.DetalleGrupoInvestigador, error) {
	// Use lowercase snake_case and $1 placeholder
	rows, err := db.Query(`SELECT idGrupo_Investigador, idGrupo, idInvestigador, rol, createdAt, updatedAt FROM Grupo_Investigador WHERE idGrupo = $1`, grupoID)
	if err != nil {
		return nil, fmt.Errorf("error querying group-investigator details by group ID: %w", err)
	}
	defer rows.Close()

	detalles := []models.DetalleGrupoInvestigador{}
	for rows.Next() {
		var d models.DetalleGrupoInvestigador
		// Ensure SELECT order matches struct fields
		if err := rows.Scan(&d.ID, &d.IDGrupo, &d.IDInvestigador, &d.Rol, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, fmt.Errorf("error scanning group-investigator detail row: %w", err)
		}
		detalles = append(detalles, d)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error after iterating through group-investigator detail rows: %w", err)
	}

	return detalles, nil
}

// DeleteDetalleGrupoInvestigador deletes a specific relationship detail by its ID.
func DeleteDetalleGrupoInvestigador(db *sql.DB, id int) error {
	// Use lowercase snake_case and $1 placeholder
	_, err := db.Exec(`DELETE FROM Grupo_Investigador WHERE idGrupo_Investigador = $1`, id)
	if err != nil {
		return fmt.Errorf("error deleting group-investigator detail: %w", err)
	}
	return nil
}

// GetDetalleGrupoInvestigadorByID retrieves a single relationship detail by its ID.
// This might be useful for updating a specific relationship (e.g., changing a role).
func GetDetalleGrupoInvestigadorByID(db *sql.DB, id int) (*models.DetalleGrupoInvestigador, error) {
	var d models.DetalleGrupoInvestigador
	// Use lowercase snake_case and $1 placeholder
	err := db.QueryRow(`SELECT idGrupo_Investigador, idGrupo, idInvestigador, rol, createdAt, updatedAt FROM Grupo_Investigador WHERE idGrupo_Investigador = $1`, id).Scan(&d.ID, &d.IDGrupo, &d.IDInvestigador, &d.Rol, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Return nil for both when not found
		}
		return nil, fmt.Errorf("error getting group-investigator detail by ID: %w", err)
	}
	return &d, nil
}

// UpdateDetalleGrupoInvestigador updates an existing relationship detail.
func UpdateDetalleGrupoInvestigador(db *sql.DB, detalle *models.DetalleGrupoInvestigador) error {
	// Use lowercase snake_case and $n placeholders
	_, err := db.Exec(`UPDATE Grupo_Investigador SET idGrupo = $1, idInvestigador = $2, rol = $3, updatedAt = CURRENT_TIMESTAMP WHERE idGrupo_Investigador = $4`, detalle.IDGrupo, detalle.IDInvestigador, detalle.Rol, detalle.ID)
	if err != nil {
		return fmt.Errorf("error updating group-investigator detail: %w", err)
	}
	return nil
}
