package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/golang-samples/run/helloworld/models"
	"github.com/GoogleCloudPlatform/golang-samples/run/helloworld/repository"
	"github.com/GoogleCloudPlatform/golang-samples/run/helloworld/utils"
	"github.com/gorilla/mux"
)

const (
	uploadDir     = "uploads"
	maxUploadSize = 10 * 1024 * 1024
	timeFormat    = "2006-01-02"
)

// Helper function to save uploaded file
func saveUploadedFile(r *http.Request, formKey string) (*string, error) {
	err := r.ParseMultipartForm(maxUploadSize)
	if err != nil {
		if err == http.ErrNotMultipart || err == http.ErrMissingFile {
			return nil, nil
		}
		return nil, fmt.Errorf("error parsing multipart form: %w", err)
	}

	file, handler, err := r.FormFile(formKey)
	if err != nil {
		if err == http.ErrMissingFile {
			return nil, nil
		}
		return nil, fmt.Errorf("error retrieving file '%s': %w", formKey, err)
	}
	defer file.Close()

	originalFilename := filepath.Base(handler.Filename)
	safeFilename := strings.ReplaceAll(originalFilename, "..", "")
	uniqueFilename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), safeFilename)

	err = os.MkdirAll(uploadDir, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("error creating upload directory: %w", err)
	}

	filePath := filepath.Join(uploadDir, uniqueFilename)
	dst, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("error creating destination file: %w", err)
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		os.Remove(filePath)
		return nil, fmt.Errorf("error copying uploaded file: %w", err)
	}

	relativePath := filepath.ToSlash(filePath)
	return &relativePath, nil
}

func removeFile(relativePath *string) error {
	if relativePath == nil || *relativePath == "" {
		return nil
	}
	err := os.Remove(*relativePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error removing file '%s': %w", *relativePath, err)
	}
	return nil
}

// GetGruposHandler handles fetching all groups or searching based on criteria with pagination.
// It *always* returns groups with their associated investigators.
func GetGruposHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Read search params
		groupName := r.URL.Query().Get("grupo")
		investigatorName := r.URL.Query().Get("investigador")
		year := r.URL.Query().Get("año")
		lineaInvestigacion := r.URL.Query().Get("lineaInvestigacion")
		tipoInvestigacion := r.URL.Query().Get("tipoInvestigacion")

		// Read pagination params
		page, limit := utils.GetPaginationParams(r)
		offset := (page - 1) * limit

		// Always expect the detailed structure
		var gruposConDetalles []models.GrupoWithInvestigadores
		var totalItems int
		var err error

		// Check if *any* search parameter is provided
		isSearch := groupName != "" || investigatorName != "" || year != "" || lineaInvestigacion != "" || tipoInvestigacion != ""

		if isSearch {
			// Perform search: returns groups with investigators and roles
			gruposConDetalles, totalItems, err = repository.SearchGrupos(db, groupName, investigatorName, year, lineaInvestigacion, tipoInvestigacion, limit, offset)
		} else {
			// Get all groups *with details* when no search parameters are present
			gruposConDetalles, totalItems, err = repository.GetAllGruposWithDetails(db, limit, offset)
		}

		if err != nil {
			log.Printf("Error getting/searching groups with details: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Calculate pagination metadata
		totalPages := 0
		if totalItems > 0 {
			totalPages = int(math.Ceil(float64(totalItems) / float64(limit)))
		}
		pagination := models.PaginationMetadata{
			TotalItems:  totalItems,
			TotalPages:  totalPages,
			CurrentPage: page,
			Limit:       limit,
		}

		// Create paginated response with the detailed data
		response := models.PaginatedResponse{
			Data:       gruposConDetalles,
			Pagination: pagination,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// GetGrupoHandler handles fetching a single group by ID.
func GetGrupoHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		idStr := vars["id"]
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid group ID", http.StatusBadRequest)
			return
		}

		grupo, err := repository.GetGrupoByID(db, id)
		if err != nil {
			log.Printf("Error getting group by ID: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if grupo == nil {
			http.Error(w, "Grupo not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(grupo)
	}
}

// CreateGrupoHandler handles creating a new group with potential file upload.
// Expects multipart/form-data
func CreateGrupoHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filePath, err := saveUploadedFile(r, "archivo")
		if err != nil {
			log.Printf("Error saving uploaded file during group creation: %v", err)
			if strings.Contains(err.Error(), "parsing multipart form") || strings.Contains(err.Error(), "request body too large") {
				http.Error(w, fmt.Sprintf("Error processing form: %v", err), http.StatusBadRequest)
			} else {
				http.Error(w, "Internal server error processing file upload", http.StatusInternalServerError)
			}
			return
		}

		var g models.Grupo
		g.Nombre = r.FormValue("nombre")
		g.NumeroResolucion = r.FormValue("numeroResolucion")
		g.LineaInvestigacion = r.FormValue("lineaInvestigacion")
		g.TipoInvestigacion = r.FormValue("tipoInvestigacion")

		fechaStr := r.FormValue("fechaRegistro")
		if fechaStr != "" {
			parsedDate, err := time.Parse(timeFormat, fechaStr)
			if err != nil {
				_ = removeFile(filePath)
				http.Error(w, fmt.Sprintf("Invalid format for fechaRegistro. Use %s", timeFormat), http.StatusBadRequest)
				return
			}
			g.FechaRegistro = parsedDate
		}

		if g.Nombre == "" || g.NumeroResolucion == "" || g.LineaInvestigacion == "" || g.TipoInvestigacion == "" {
			_ = removeFile(filePath)
			http.Error(w, "Missing required text fields: nombre, numeroResolucion, lineaInvestigacion, tipoInvestigacion", http.StatusBadRequest)
			return
		}
		if g.FechaRegistro.IsZero() {
			_ = removeFile(filePath)
			http.Error(w, fmt.Sprintf("Missing or invalid required field: fechaRegistro (use format %s)", timeFormat), http.StatusBadRequest)
			return
		}

		g.Archivo = filePath

		if err := repository.CreateGrupo(db, &g); err != nil {
			log.Printf("Error creating group in repository: %v", err)
			_ = removeFile(filePath)
			http.Error(w, "Internal server error saving group", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(g)
	}
}

// UpdateGrupoHandler handles updating an existing group, potentially replacing the file.
// Expects multipart/form-data
func UpdateGrupoHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		idStr := vars["id"]
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid group ID", http.StatusBadRequest)
			return
		}

		existingGrupo, err := repository.GetGrupoByID(db, id)
		if err != nil {
			log.Printf("Error getting group by ID for update: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if existingGrupo == nil {
			http.Error(w, "Grupo not found for update", http.StatusNotFound)
			return
		}

		newFilePath, err := saveUploadedFile(r, "archivo")
		if err != nil {
			log.Printf("Error saving uploaded file during group update: %v", err)
			if strings.Contains(err.Error(), "parsing multipart form") || strings.Contains(err.Error(), "request body too large") {
				http.Error(w, fmt.Sprintf("Error processing form: %v", err), http.StatusBadRequest)
			} else {
				http.Error(w, "Internal server error processing file upload", http.StatusInternalServerError)
			}
			return
		}

		var updatedGrupo models.Grupo
		updatedGrupo.ID = id
		updatedGrupo.Nombre = r.FormValue("nombre")
		updatedGrupo.NumeroResolucion = r.FormValue("numeroResolucion")
		updatedGrupo.LineaInvestigacion = r.FormValue("lineaInvestigacion")
		updatedGrupo.TipoInvestigacion = r.FormValue("tipoInvestigacion")

		fechaStr := r.FormValue("fechaRegistro")
		if fechaStr != "" {
			parsedDate, err := time.Parse(timeFormat, fechaStr)
			if err != nil {
				_ = removeFile(newFilePath)
				http.Error(w, fmt.Sprintf("Invalid format for fechaRegistro. Use %s", timeFormat), http.StatusBadRequest)
				return
			}
			updatedGrupo.FechaRegistro = parsedDate
		} else {
			updatedGrupo.FechaRegistro = existingGrupo.FechaRegistro
		}

		if updatedGrupo.Nombre == "" {
			updatedGrupo.Nombre = existingGrupo.Nombre
		}
		if updatedGrupo.NumeroResolucion == "" {
			updatedGrupo.NumeroResolucion = existingGrupo.NumeroResolucion
		}
		if updatedGrupo.LineaInvestigacion == "" {
			updatedGrupo.LineaInvestigacion = existingGrupo.LineaInvestigacion
		}
		if updatedGrupo.TipoInvestigacion == "" {
			updatedGrupo.TipoInvestigacion = existingGrupo.TipoInvestigacion
		}

		var oldFilePathToDelete *string = nil
		if newFilePath != nil {
			updatedGrupo.Archivo = newFilePath
			if existingGrupo.Archivo != nil && *existingGrupo.Archivo != "" && *existingGrupo.Archivo != *newFilePath {
				oldFilePathToDelete = existingGrupo.Archivo
			}
		} else {
			updatedGrupo.Archivo = existingGrupo.Archivo
		}

		if err := repository.UpdateGrupo(db, &updatedGrupo); err != nil {
			log.Printf("Error updating group in repository: %v", err)
			_ = removeFile(newFilePath)
			http.Error(w, "Internal server error updating group", http.StatusInternalServerError)
			return
		}

		if oldFilePathToDelete != nil {
			err := removeFile(oldFilePathToDelete)
			if err != nil {
				log.Printf("Warning: Error deleting old file '%s' after group update: %v", *oldFilePathToDelete, err)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(updatedGrupo)
	}
}

// DeleteGrupoHandler handles deleting a group by ID.
func DeleteGrupoHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		idStr := vars["id"]
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid group ID", http.StatusBadRequest)
			return
		}

		if err := repository.DeleteGrupo(db, id); err != nil {
			log.Printf("Error deleting group: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// GetGrupoDetailsHandler retrieves a group's details along with its associated investigators.
func GetGrupoDetailsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		idStr := vars["id"]
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid group ID", http.StatusBadRequest)
			return
		}

		grupoWithInvestigadores, err := repository.GetGrupoDetails(db, id)
		if err != nil {
			log.Printf("Error getting group details from repository: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if grupoWithInvestigadores == nil {
			http.Error(w, "Grupo not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(grupoWithInvestigadores)
	}
}

// Struct to represent the investigator relationship in the combined creation request
type InvestigatorRelationshipRequest struct {
	IDInvestigador int    `json:"idInvestigador"`
	TipoRelacion   string `json:"tipoRelacion"`
}

// Struct to represent the combined group and details creation request body
type CreateGrupoWithDetailsRequest struct {
	models.Grupo   `json:"grupo"`
	Investigadores []InvestigatorRelationshipRequest `json:"investigadores"`
}

// Handler for creating a group with associated investigator details
func CreateGrupoWithDetailsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var requestBody CreateGrupoWithDetailsRequest
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Start a transaction
		tx, err := db.Begin()
		if err != nil {
			log.Printf("Error starting transaction: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		// Use a deferred function for commit/rollback based on error
		defer func() {
			if p := recover(); p != nil {
				tx.Rollback()
				panic(p) // Re-panic after rollback
			} else if err != nil {
				// Log the error that caused the rollback
				log.Printf("Rolling back transaction due to error: %v", err)
				tx.Rollback() // Rollback on any error
			} else {
				err = tx.Commit() // Commit otherwise
				if err != nil {
					log.Printf("Error committing transaction: %v", err)
					// Don't send HTTP error here as response might have already been written
				}
			}
		}()

		// Create the group within the transaction using QueryRow with RETURNING
		grupoToCreate := requestBody.Grupo
		// Use lowercase snake_case names and $n placeholders
		groupInsertQuery := `INSERT INTO grupo (nombre, numeroResolucion, lineaInvestigacion, tipoInvestigacion, fechaRegistro, archivo) VALUES ($1, $2, $3, $4, $5, $6) RETURNING idGrupo`
		var grupoID int64 // Use int64 for Scan with RETURNING

		err = tx.QueryRow(groupInsertQuery, grupoToCreate.Nombre, grupoToCreate.NumeroResolucion, grupoToCreate.LineaInvestigacion, grupoToCreate.TipoInvestigacion, grupoToCreate.FechaRegistro, grupoToCreate.Archivo).Scan(&grupoID)
		if err != nil {
			// Error is logged and transaction rolled back by defer
			log.Printf("Error inserting group in transaction: %v", err)
			http.Error(w, "Internal server error during group creation", http.StatusInternalServerError)
			return
		}

		// Create the detailed relationships within the transaction using Exec
		// Use lowercase snake_case names and $n placeholders
		detailInsertQuery := `INSERT INTO Grupo_Investigador (idGrupo, idInvestigador, tipo_relacion) VALUES ($1, $2, $3)`
		for _, invRel := range requestBody.Investigadores {
			_, err = tx.Exec(detailInsertQuery, grupoID, invRel.IDInvestigador, invRel.TipoRelacion)
			if err != nil {
				// Error is logged and transaction rolled back by defer
				log.Printf("Error inserting group-investigator detail in transaction: %v", err)
				http.Error(w, "Internal server error during detail creation", http.StatusInternalServerError)
				return
			}
		}

		// If we reach here without error, the defer func will handle the commit.

		// Prepare the response
		grupoToCreate.ID = int(grupoID) // Convert int64 back to int for the response model
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(grupoToCreate)
	}
}

// GetGruposByInvestigadorHandler maneja la obtención de todos los grupos a los que pertenece un investigador.
func GetGruposByInvestigadorHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		idStr := vars["idInvestigador"]
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "ID de investigador inválido", http.StatusBadRequest)
			return
		}

		gruposConIntegrantes, err := repository.GetGruposByInvestigadorID(db, id)
		if err != nil {
			log.Printf("Error obteniendo grupos por investigador: %v", err)
			http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
			return
		}

		// Enriquecer la respuesta para incluir los integrantes con su rol
		var respuesta []map[string]interface{}
		for _, grupoConInt := range gruposConIntegrantes {
			respuesta = append(respuesta, map[string]interface{}{
				"grupo":       grupoConInt["grupo"],
				"integrantes": grupoConInt["integrantes"],
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(respuesta)
	}
}

// GetAllGruposWithDetailsHandler retrieves all groups with their associated investigators and roles, paginated.
func GetAllGruposWithDetailsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Read pagination params
		page, limit := utils.GetPaginationParams(r)
		offset := (page - 1) * limit

		// Call the repository function to get all groups with details
		gruposConDetalles, totalItems, err := repository.GetAllGruposWithDetails(db, limit, offset)
		if err != nil {
			log.Printf("Error getting all groups with details: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Calculate pagination metadata
		totalPages := 0
		if totalItems > 0 {
			totalPages = int(math.Ceil(float64(totalItems) / float64(limit)))
		}
		pagination := models.PaginationMetadata{
			TotalItems:  totalItems,
			TotalPages:  totalPages,
			CurrentPage: page,
			Limit:       limit,
		}

		// Create paginated response
		response := models.PaginatedResponse{
			Data:       gruposConDetalles, // Data is []GrupoWithInvestigadores
			Pagination: pagination,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
