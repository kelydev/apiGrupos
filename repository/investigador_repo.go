package repository

import (
	"database/sql"
	"fmt"
	"strings" // Import strings for query building

	"github.com/GoogleCloudPlatform/golang-samples/run/helloworld/models"
)

// GetAllInvestigadores retrieves a paginated list of all investigators.
func GetAllInvestigadores(db *sql.DB, limit, offset int) ([]models.Investigador, int, error) {
	// Query for the data page
	query := `SELECT idInvestigador, nombre, apellido, createdAt, updatedAt FROM investigador ORDER BY nombre, apellido LIMIT $1 OFFSET $2`
	rows, err := db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("error querying investigators page: %w", err)
	}
	defer rows.Close()

	investigadores := []models.Investigador{}
	for rows.Next() {
		var inv models.Investigador
		if err := rows.Scan(&inv.ID, &inv.Nombre, &inv.Apellido, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("error scanning investigator row: %w", err)
		}
		investigadores = append(investigadores, inv)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error after iterating through investigator rows: %w", err)
	}

	// Query for the total count
	var total int
	countQuery := `SELECT COUNT(*) FROM investigador`
	if err := db.QueryRow(countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("error querying total investigator count: %w", err)
	}

	return investigadores, total, nil
}

// GetInvestigadorByID retrieves a single investigator by their ID.
func GetInvestigadorByID(db *sql.DB, id int) (*models.Investigador, error) {
	var inv models.Investigador
	err := db.QueryRow(`SELECT idInvestigador, nombre, apellido, createdAt, updatedAt FROM investigador WHERE idInvestigador = $1`, id).Scan(&inv.ID, &inv.Nombre, &inv.Apellido, &inv.CreatedAt, &inv.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Return nil for both when not found
		}
		return nil, fmt.Errorf("error getting investigator by ID: %w", err)
	}
	return &inv, nil
}

// CreateInvestigador inserts a new investigator into the database.
func CreateInvestigador(db *sql.DB, inv *models.Investigador) error {
	query := `INSERT INTO investigador (nombre, apellido) VALUES ($1, $2) RETURNING idInvestigador, createdAt, updatedAt`
	err := db.QueryRow(query, inv.Nombre, inv.Apellido).Scan(&inv.ID, &inv.CreatedAt, &inv.UpdatedAt)
	if err != nil {
		return fmt.Errorf("error inserting investigator: %w", err)
	}
	return nil
}

// UpdateInvestigador updates an existing investigator in the database.
func UpdateInvestigador(db *sql.DB, inv *models.Investigador) error {
	_, err := db.Exec(`UPDATE investigador SET nombre = $1, apellido = $2, updatedAt = CURRENT_TIMESTAMP WHERE idInvestigador = $3`, inv.Nombre, inv.Apellido, inv.ID)
	if err != nil {
		return fmt.Errorf("error updating investigator: %w", err)
	}
	return nil
}

// DeleteInvestigador deletes an investigator from the database.
func DeleteInvestigador(db *sql.DB, id int) error {
	_, err := db.Exec(`DELETE FROM investigador WHERE idInvestigador = $1`, id)
	if err != nil {
		return fmt.Errorf("error deleting investigator: %w", err)
	}
	return nil
}

// SearchInvestigadores searches for investigators with pagination.
func SearchInvestigadores(db *sql.DB, name string, limit, offset int) ([]models.Investigador, int, error) {
	// Base query and conditions
	baseQuery := `FROM investigador WHERE 1=1`
	var conditions []string
	args := []interface{}{}
	placeholderCount := 1

	if name != "" {
		conditions = append(conditions, fmt.Sprintf(`(unaccent(nombre) ILIKE unaccent($%d) OR unaccent(apellido) ILIKE unaccent($%d))`, placeholderCount, placeholderCount+1))
		searchPattern := "%" + name + "%"
		args = append(args, searchPattern, searchPattern)
		placeholderCount += 2
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " AND " + strings.Join(conditions, " AND ")
	}

	// Query for the data page
	query := fmt.Sprintf(`SELECT idInvestigador, nombre, apellido, createdAt, updatedAt %s %s ORDER BY nombre, apellido LIMIT $%d OFFSET $%d`, baseQuery, whereClause, placeholderCount, placeholderCount+1)
	finalArgs := append(args, limit, offset)
	rows, err := db.Query(query, finalArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("error searching investigators page: %w", err)
	}
	defer rows.Close()

	investigadores := []models.Investigador{}
	for rows.Next() {
		var inv models.Investigador
		if err := rows.Scan(&inv.ID, &inv.Nombre, &inv.Apellido, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("error scanning investigator row during search: %w", err)
		}
		investigadores = append(investigadores, inv)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error after iterating through investigator search rows: %w", err)
	}

	// Query for the total count with the same filters
	var total int
	countQuery := fmt.Sprintf(`SELECT COUNT(*) %s %s`, baseQuery, whereClause)
	if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil { // Use original args for count
		return nil, 0, fmt.Errorf("error searching total investigator count: %w", err)
	}

	return investigadores, total, nil
}

// GetAllInvestigadoresNoPagination retrieves ALL investigators without pagination.
func GetAllInvestigadoresNoPagination(db *sql.DB) ([]models.Investigador, error) {
	query := `SELECT idInvestigador, nombre, apellido, createdAt, updatedAt FROM investigador ORDER BY nombre, apellido`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error querying all investigators: %w", err)
	}
	defer rows.Close()

	investigadores := []models.Investigador{}
	for rows.Next() {
		var inv models.Investigador
		if err := rows.Scan(&inv.ID, &inv.Nombre, &inv.Apellido, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
			return nil, fmt.Errorf("error scanning investigator row (no pagination): %w", err)
		}
		investigadores = append(investigadores, inv)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error after iterating through all investigator rows: %w", err)
	}

	return investigadores, nil
}
