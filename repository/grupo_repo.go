package repository

import (
	"database/sql"
	"fmt"
	"strings"

	// Import math for ceiling calculation
	"github.com/GoogleCloudPlatform/golang-samples/run/helloworld/models"
)

// GetAllGrupos retrieves a paginated list of all groups.
func GetAllGrupos(db *sql.DB, limit, offset int) ([]models.Grupo, int, error) {
	// Query for the data page
	query := `SELECT idGrupo, nombre, numeroResolucion, lineaInvestigacion, tipoInvestigacion, fechaRegistro, archivo, createdAt, updatedAt FROM grupo ORDER BY nombre LIMIT $1 OFFSET $2`
	rows, err := db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("error querying groups page: %w", err)
	}
	defer rows.Close()

	grupos := []models.Grupo{}
	for rows.Next() {
		var g models.Grupo
		if err := rows.Scan(&g.ID, &g.Nombre, &g.NumeroResolucion, &g.LineaInvestigacion, &g.TipoInvestigacion, &g.FechaRegistro, &g.Archivo, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("error scanning group row: %w", err)
		}
		grupos = append(grupos, g)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error after iterating through group rows: %w", err)
	}

	// Query for the total count
	var total int
	countQuery := `SELECT COUNT(*) FROM grupo`
	if err := db.QueryRow(countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("error querying total group count: %w", err)
	}

	return grupos, total, nil
}

// GetGrupoByID retrieves a single group by its ID.
func GetGrupoByID(db *sql.DB, id int) (*models.Grupo, error) {
	var g models.Grupo
	err := db.QueryRow(`SELECT idGrupo, nombre, numeroResolucion, lineaInvestigacion, tipoInvestigacion, fechaRegistro, archivo, createdAt, updatedAt FROM grupo WHERE idGrupo = $1`, id).Scan(&g.ID, &g.Nombre, &g.NumeroResolucion, &g.LineaInvestigacion, &g.TipoInvestigacion, &g.FechaRegistro, &g.Archivo, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Return nil for both when not found
		}
		return nil, fmt.Errorf("error getting group by ID: %w", err)
	}
	return &g, nil
}

// CreateGrupo inserts a new group into the database.
func CreateGrupo(db *sql.DB, g *models.Grupo) error {
	query := `INSERT INTO grupo (nombre, numeroResolucion, lineaInvestigacion, tipoInvestigacion, fechaRegistro, archivo) VALUES ($1, $2, $3, $4, $5, $6) RETURNING idGrupo, createdAt, updatedAt`
	err := db.QueryRow(query, g.Nombre, g.NumeroResolucion, g.LineaInvestigacion, g.TipoInvestigacion, g.FechaRegistro, g.Archivo).Scan(&g.ID, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return fmt.Errorf("error inserting group: %w", err)
	}
	return nil
}

// UpdateGrupo updates an existing group in the database.
func UpdateGrupo(db *sql.DB, g *models.Grupo) error {
	_, err := db.Exec(`UPDATE grupo SET nombre = $1, numeroResolucion = $2, lineaInvestigacion = $3, tipoInvestigacion = $4, fechaRegistro = $5, archivo = $6, updatedAt = CURRENT_TIMESTAMP WHERE idGrupo = $7`, g.Nombre, g.NumeroResolucion, g.LineaInvestigacion, g.TipoInvestigacion, g.FechaRegistro, g.Archivo, g.ID)
	if err != nil {
		return fmt.Errorf("error updating group: %w", err)
	}
	return nil
}

// DeleteGrupo deletes a group from the database.
func DeleteGrupo(db *sql.DB, id int) error {
	_, err := db.Exec(`DELETE FROM grupo WHERE idGrupo = $1`, id)
	if err != nil {
		return fmt.Errorf("error deleting group: %w", err)
	}
	return nil
}

// SearchGrupos searches for groups with pagination and returns them with investigators and roles.
func SearchGrupos(db *sql.DB, groupName, investigatorName, year, lineaInvestigacion, tipoInvestigacion string, limit, offset int) ([]models.GrupoWithInvestigadores, int, error) {
	args := []interface{}{}
	placeholderCount := 1

	// --- Build WHERE clause dynamically (for the initial filtering CTE) ---
	whereConditions := ""

	if groupName != "" {
		whereConditions += fmt.Sprintf(` AND unaccent(g.nombre) ILIKE unaccent($%d)`, placeholderCount)
		args = append(args, "%"+groupName+"%")
		placeholderCount++
	}

	if investigatorName != "" {
		whereConditions += fmt.Sprintf(` AND unaccent(i.nombre || ' ' || i.apellido) ILIKE unaccent($%d)`, placeholderCount)
		args = append(args, "%"+investigatorName+"%")
		placeholderCount++
	}

	if year != "" {
		whereConditions += fmt.Sprintf(` AND EXTRACT(YEAR FROM g.fechaRegistro) = $%d`, placeholderCount)
		args = append(args, year)
		placeholderCount++
	}

	if lineaInvestigacion != "" {
		whereConditions += fmt.Sprintf(` AND unaccent(g.lineaInvestigacion) ILIKE unaccent($%d)`, placeholderCount)
		args = append(args, "%"+lineaInvestigacion+"%")
		placeholderCount++
	}

	if tipoInvestigacion != "" {
		whereConditions += fmt.Sprintf(` AND unaccent(g.tipoInvestigacion) ILIKE unaccent($%d)`, placeholderCount)
		args = append(args, "%"+tipoInvestigacion+"%")
		placeholderCount++
	}
	// --- End WHERE clause build ---

	// CTE 1: Find all unique group IDs matching the filters
	cteFilteredGroups := `
	WITH FilteredGroups AS (
		SELECT DISTINCT g.idGrupo
		FROM grupo g
		LEFT JOIN Grupo_Investigador dgi ON g.idGrupo = dgi.idGrupo
		LEFT JOIN investigador i ON dgi.idInvestigador = i.idInvestigador
		WHERE 1=1` + whereConditions + `
	)`

	// --- Query for the total count using the first CTE ---
	var totalItems int
	countQuery := cteFilteredGroups + ` SELECT COUNT(*) FROM FilteredGroups`
	if err := db.QueryRow(countQuery, args...).Scan(&totalItems); err != nil { // Use original args for count
		return nil, 0, fmt.Errorf("error searching total group count: %w", err)
	}

	// If no items found, return early
	if totalItems == 0 {
		return []models.GrupoWithInvestigadores{}, 0, nil
	}

	// --- Build the final query to get paginated details ---

	// CTE 2: Paginate the filtered group IDs
	ctePaginatedIDs := fmt.Sprintf(`,
	PaginatedGroupIDs AS (
		SELECT idGrupo
		FROM FilteredGroups
		ORDER BY idGrupo -- Or another relevant field like g.nombre from the join if needed
		LIMIT $%d OFFSET $%d
	)`, placeholderCount, placeholderCount+1)

	// Main query to get details for the paginated group IDs
	dataQuery := cteFilteredGroups + ctePaginatedIDs + `
	SELECT
		g.idGrupo, g.nombre, g.numeroResolucion, g.lineaInvestigacion, g.tipoInvestigacion, g.fechaRegistro, g.archivo, g.createdAt, g.updatedAt,
		i.idInvestigador, i.nombre as invNombre, i.apellido as invApellido, i.createdAt as invCreatedAt, i.updatedAt as invUpdatedAt,
		dgi.rol
	FROM grupo g
	LEFT JOIN Grupo_Investigador dgi ON g.idGrupo = dgi.idGrupo
	LEFT JOIN investigador i ON dgi.idInvestigador = i.idInvestigador
	WHERE g.idGrupo IN (SELECT idGrupo FROM PaginatedGroupIDs)
	ORDER BY g.idGrupo, i.idInvestigador -- Ensure consistent order for grouping`

	// Append limit and offset to the original args
	finalArgs := append(args, limit, offset)
	rows, err := db.Query(dataQuery, finalArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("error searching groups page with details: %w, Query: %s, Args: %v", err, dataQuery, finalArgs)
	}
	defer rows.Close()

	// --- Process rows and group investigators ---
	grupoMap := make(map[int]*models.GrupoWithInvestigadores)
	// Slice to maintain order based on PaginatedGroupIDs query order
	orderedGrupos := []*models.GrupoWithInvestigadores{}

	for rows.Next() {
		var g models.Grupo
		var invID sql.NullInt64 // Use Null types for LEFT JOIN results
		var invNombre, invApellido, invRol sql.NullString
		var invCreatedAt, invUpdatedAt sql.NullTime

		if err := rows.Scan(
			&g.ID, &g.Nombre, &g.NumeroResolucion, &g.LineaInvestigacion, &g.TipoInvestigacion, &g.FechaRegistro, &g.Archivo, &g.CreatedAt, &g.UpdatedAt,
			&invID, &invNombre, &invApellido, &invCreatedAt, &invUpdatedAt,
			&invRol,
		); err != nil {
			return nil, 0, fmt.Errorf("error scanning group/investigator row during search: %w", err)
		}

		// Check if we've already seen this group
		grupoWithDetails, exists := grupoMap[g.ID]
		if !exists {
			// First time seeing this group (within the paginated set)
			grupoWithDetails = &models.GrupoWithInvestigadores{
				Grupo:          g,
				Investigadores: []models.InvestigadorConRol{}, // Initialize empty slice
			}
			grupoMap[g.ID] = grupoWithDetails
			orderedGrupos = append(orderedGrupos, grupoWithDetails) // Add to ordered list
		}

		// If an investigator was joined (not a group without investigators matched by filter)
		if invID.Valid {
			inv := models.InvestigadorConRol{
				ID:       int(invID.Int64),
				Nombre:   invNombre.String,
				Apellido: invApellido.String,
				Rol:      invRol.String,
			}
			if invCreatedAt.Valid {
				inv.CreatedAt = invCreatedAt.Time
			}
			if invUpdatedAt.Valid {
				inv.UpdatedAt = invUpdatedAt.Time
			}
			// Append investigator only if valid
			grupoMap[g.ID].Investigadores = append(grupoMap[g.ID].Investigadores, inv)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error after iterating through group search rows: %w", err)
	}

	// Convert []*models.GrupoWithInvestigadores to []models.GrupoWithInvestigadores
	result := make([]models.GrupoWithInvestigadores, len(orderedGrupos))
	for i, ptr := range orderedGrupos {
		result[i] = *ptr
	}

	return result, totalItems, nil
}

// GetGrupoDetails retrieves a group and its associated investigators including their roles.
func GetGrupoDetails(db *sql.DB, id int) (*models.GrupoWithInvestigadores, error) {
	// 1. Get the group details
	grupo, err := GetGrupoByID(db, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("error in GetGrupoByID called from GetGrupoDetails: %w", err)
	}
	if grupo == nil { // Should not happen
		return nil, nil
	}

	// 2. Get associated investigators with their roles in this specific group
	query := `
		SELECT i.idInvestigador, i.nombre, i.apellido, dgi.rol, i.createdAt, i.updatedAt
		FROM investigador i
		JOIN Grupo_Investigador dgi ON i.idInvestigador = dgi.idInvestigador
		WHERE dgi.idGrupo = $1
	`
	rows, err := db.Query(query, id)
	if err != nil {
		return nil, fmt.Errorf("error querying investigators for group details: %w", err)
	}
	defer rows.Close()

	investigadores := []models.InvestigadorConRol{}
	for rows.Next() {
		var inv models.InvestigadorConRol
		// Scan id, nombre, apellido, rol, createdAt, updatedAt
		if err := rows.Scan(&inv.ID, &inv.Nombre, &inv.Apellido, &inv.Rol, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
			return nil, fmt.Errorf("error scanning investigator row with role for group details: %w", err)
		}
		investigadores = append(investigadores, inv)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error after iterating investigator rows for group details: %w", err)
	}

	// 3. Combine results
	grupoDetail := &models.GrupoWithInvestigadores{
		Grupo:          *grupo,
		Investigadores: investigadores, // Now contains investigators with roles
	}

	return grupoDetail, nil
}

// GetGruposByInvestigadorID obtiene todos los grupos a los que pertenece un investigador dado su id.
func GetGruposByInvestigadorID(db *sql.DB, idInvestigador int) ([]map[string]interface{}, error) {
	query := `SELECT g.idGrupo, g.nombre, g.numeroResolucion, g.lineaInvestigacion, g.tipoInvestigacion, g.fechaRegistro, g.archivo, g.createdAt, g.updatedAt
				 , dgi.rol
			 FROM grupo g
			 JOIN Grupo_Investigador dgi ON g.idGrupo = dgi.idGrupo
			 WHERE dgi.idInvestigador = $1`
	rows, err := db.Query(query, idInvestigador)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo grupos por idInvestigador: %w", err)
	}
	defer rows.Close()

	var gruposConIntegrantes []map[string]interface{}
	for rows.Next() {
		var g models.Grupo
		var rol string
		if err := rows.Scan(&g.ID, &g.Nombre, &g.NumeroResolucion, &g.LineaInvestigacion, &g.TipoInvestigacion, &g.FechaRegistro, &g.Archivo, &g.CreatedAt, &g.UpdatedAt, &rol); err != nil {
			return nil, fmt.Errorf("error escaneando grupo: %w", err)
		}

		// Obtener los integrantes y sus roles para este grupo
		queryIntegrantes := `SELECT i.idInvestigador, i.nombre, i.apellido, dgi.rol
			FROM investigador i
			JOIN Grupo_Investigador dgi ON i.idInvestigador = dgi.idInvestigador
			WHERE dgi.idGrupo = $1`
		rowsIntegrantes, err := db.Query(queryIntegrantes, g.ID)
		if err != nil {
			return nil, fmt.Errorf("error obteniendo integrantes del grupo: %w", err)
		}
		var integrantesConRol []map[string]interface{}
		for rowsIntegrantes.Next() {
			var idInvestigador int
			var nombre, apellido, rolIntegrante string
			if err := rowsIntegrantes.Scan(&idInvestigador, &nombre, &apellido, &rolIntegrante); err != nil {
				rowsIntegrantes.Close()
				return nil, fmt.Errorf("error escaneando integrante: %w", err)
			}
			integrantesConRol = append(integrantesConRol, map[string]interface{}{
				"idInvestigador": idInvestigador,
				"nombre":         nombre,
				"apellido":       apellido,
				"rol":            rolIntegrante,
			})
		}
		rowsIntegrantes.Close()

		grupoMap := map[string]interface{}{
			"grupo":       g,
			"integrantes": integrantesConRol,
		}
		gruposConIntegrantes = append(gruposConIntegrantes, grupoMap)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error después de iterar los grupos: %w", err)
	}
	return gruposConIntegrantes, nil
}

// GetAllGruposWithDetails retrieves a paginated list of all groups with their associated investigators and roles.
func GetAllGruposWithDetails(db *sql.DB, limit, offset int) ([]models.GrupoWithInvestigadores, int, error) {
	// 1. Get the total count of groups
	var totalItems int
	countQuery := `SELECT COUNT(*) FROM grupo`
	if err := db.QueryRow(countQuery).Scan(&totalItems); err != nil {
		return nil, 0, fmt.Errorf("error querying total group count for get all with details: %w", err)
	}

	// If no groups, return early
	if totalItems == 0 {
		return []models.GrupoWithInvestigadores{}, 0, nil
	}

	// 2. Get the IDs of the groups for the current page
	paginatedIDsQuery := `SELECT idGrupo FROM grupo ORDER BY nombre, idGrupo LIMIT $1 OFFSET $2`
	rowsIDs, err := db.Query(paginatedIDsQuery, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("error querying paginated group IDs: %w", err)
	}
	defer rowsIDs.Close()

	var groupIDs []interface{} // Use interface{} for IN clause argument
	var groupIDOrder []int     // Maintain the order for final result sorting
	for rowsIDs.Next() {
		var id int
		if err := rowsIDs.Scan(&id); err != nil {
			return nil, 0, fmt.Errorf("error scanning group ID: %w", err)
		}
		groupIDs = append(groupIDs, id)
		groupIDOrder = append(groupIDOrder, id)
	}
	if err := rowsIDs.Err(); err != nil {
		return nil, 0, fmt.Errorf("error after iterating group IDs: %w", err)
	}

	// If no IDs found for this page (shouldn't happen if totalItems > 0 and offset is valid, but check anyway)
	if len(groupIDs) == 0 {
		return []models.GrupoWithInvestigadores{}, totalItems, nil
	}

	// 3. Get details for the selected group IDs using LEFT JOINs
	// Build the placeholder string for the IN clause ($1, $2, $3...)
	placeholders := make([]string, len(groupIDs))
	for i := range placeholders {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}
	placeholderString := fmt.Sprintf("(%s)", strings.Join(placeholders, ", ")) // Generates ($1, $2, ...)

	detailsQuery := `
	SELECT
		g.idGrupo, g.nombre, g.numeroResolucion, g.lineaInvestigacion, g.tipoInvestigacion, g.fechaRegistro, g.archivo, g.createdAt, g.updatedAt,
		i.idInvestigador, i.nombre as invNombre, i.apellido as invApellido, i.createdAt as invCreatedAt, i.updatedAt as invUpdatedAt,
		dgi.rol
	FROM grupo g
	LEFT JOIN Grupo_Investigador dgi ON g.idGrupo = dgi.idGrupo
	LEFT JOIN investigador i ON dgi.idInvestigador = i.idInvestigador
	WHERE g.idGrupo IN ` + placeholderString + `
	ORDER BY g.nombre, g.idGrupo, invApellido, invNombre -- Consistent ordering is important for grouping` // Order matching the ID query helps, but Go map iteration isn't ordered

	rowsDetails, err := db.Query(detailsQuery, groupIDs...) // Pass IDs as variadic arguments
	if err != nil {
		return nil, 0, fmt.Errorf("error querying group details for selected IDs: %w, Query: %s, Args: %v", err, detailsQuery, groupIDs)
	}
	defer rowsDetails.Close()

	// 4. Group results in Go
	grupoMap := make(map[int]*models.GrupoWithInvestigadores)

	for rowsDetails.Next() {
		var g models.Grupo
		var invID sql.NullInt64
		var invNombre, invApellido, invRol sql.NullString
		var invCreatedAt, invUpdatedAt sql.NullTime

		if err := rowsDetails.Scan(
			&g.ID, &g.Nombre, &g.NumeroResolucion, &g.LineaInvestigacion, &g.TipoInvestigacion, &g.FechaRegistro, &g.Archivo, &g.CreatedAt, &g.UpdatedAt,
			&invID, &invNombre, &invApellido, &invCreatedAt, &invUpdatedAt,
			&invRol,
		); err != nil {
			return nil, 0, fmt.Errorf("error scanning group/investigator row during get all with details: %w", err)
		}

		// Check if we've already seen this group
		grupoWithDetails, exists := grupoMap[g.ID]
		if !exists {
			grupoWithDetails = &models.GrupoWithInvestigadores{
				Grupo:          g,
				Investigadores: []models.InvestigadorConRol{},
			}
			grupoMap[g.ID] = grupoWithDetails
		}

		// If an investigator was joined, add them
		if invID.Valid {
			inv := models.InvestigadorConRol{
				ID:       int(invID.Int64),
				Nombre:   invNombre.String,
				Apellido: invApellido.String,
				Rol:      invRol.String,
			}
			if invCreatedAt.Valid {
				inv.CreatedAt = invCreatedAt.Time
			}
			if invUpdatedAt.Valid {
				inv.UpdatedAt = invUpdatedAt.Time
			}
			// Avoid adding duplicates if the DB somehow returns multiple identical rows (shouldn't happen with proper schema)
			found := false
			for _, existingInv := range grupoWithDetails.Investigadores {
				if existingInv.ID == inv.ID {
					found = true
					break
				}
			}
			if !found {
				grupoWithDetails.Investigadores = append(grupoWithDetails.Investigadores, inv)
			}
		}
	}

	if err := rowsDetails.Err(); err != nil {
		return nil, 0, fmt.Errorf("error after iterating through get all groups with details rows: %w", err)
	}

	// 5. Build the final result slice, respecting the paginated order
	result := make([]models.GrupoWithInvestigadores, 0, len(groupIDOrder))
	for _, id := range groupIDOrder {
		if grupoData, ok := grupoMap[id]; ok {
			result = append(result, *grupoData)
		}
		// If a group ID was selected but somehow not found in the details query (shouldn't happen), it's skipped.
	}

	return result, totalItems, nil
}
