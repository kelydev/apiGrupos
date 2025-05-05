package controllers

import (
	"database/sql"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strconv"

	"github.com/GoogleCloudPlatform/golang-samples/run/helloworld/models"
	"github.com/GoogleCloudPlatform/golang-samples/run/helloworld/repository"
	"github.com/GoogleCloudPlatform/golang-samples/run/helloworld/utils"
	"github.com/gorilla/mux"
)

// GetInvestigadoresHandler handles fetching all investigators or searching by name with pagination.
func GetInvestigadoresHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		page, limit := utils.GetPaginationParams(r)
		offset := (page - 1) * limit

		var investigadores []models.Investigador
		var totalItems int
		var err error

		if name != "" {
			investigadores, totalItems, err = repository.SearchInvestigadores(db, name, limit, offset)
		} else {
			investigadores, totalItems, err = repository.GetAllInvestigadores(db, limit, offset)
		}

		if err != nil {
			log.Printf("Error getting/searching investigators: %v", err)
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
			Data:       investigadores,
			Pagination: pagination,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// GetInvestigadorHandler handles fetching a single investigator by ID.
func GetInvestigadorHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		idStr := vars["id"]
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid investigator ID", http.StatusBadRequest)
			return
		}

		investigador, err := repository.GetInvestigadorByID(db, id)
		if err != nil {
			log.Printf("Error getting investigator by ID: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if investigador == nil {
			http.Error(w, "Investigador not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(investigador)
	}
}

// CreateInvestigadorHandler handles creating a new investigator.
func CreateInvestigadorHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var inv models.Investigador
		if err := json.NewDecoder(r.Body).Decode(&inv); err != nil {
			// Consider logging the actual error for debugging
			// log.Printf("Error decoding investigator JSON: %v", err)
			http.Error(w, "Invalid request body format", http.StatusBadRequest)
			return
		}

		// --- VALIDACIÓN ---
		if inv.Nombre == "" || inv.Apellido == "" {
			http.Error(w, "Missing required fields: nombre and apellido", http.StatusBadRequest)
			return
		}
		// --- FIN VALIDACIÓN ---

		if err := repository.CreateInvestigador(db, &inv); err != nil {
			log.Printf("Error creating investigator: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(inv)
	}
}

// UpdateInvestigadorHandler handles updating an existing investigator.
func UpdateInvestigadorHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		idStr := vars["id"]
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid investigator ID", http.StatusBadRequest)
			return
		}

		var inv models.Investigador
		if err := json.NewDecoder(r.Body).Decode(&inv); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Ensure the ID in the body matches the ID in the URL
		inv.ID = id

		if err := repository.UpdateInvestigador(db, &inv); err != nil {
			log.Printf("Error updating investigator: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(inv)
	}
}

// DeleteInvestigadorHandler handles deleting an investigator by ID.
func DeleteInvestigadorHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		idStr := vars["id"]
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid investigator ID", http.StatusBadRequest)
			return
		}

		if err := repository.DeleteInvestigador(db, id); err != nil {
			log.Printf("Error deleting investigator: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// GetAllInvestigadoresNoPaginationHandler handles fetching ALL investigators without pagination.
func GetAllInvestigadoresNoPaginationHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		investigadores, err := repository.GetAllInvestigadoresNoPagination(db)
		if err != nil {
			log.Printf("Error getting all investigators (no pagination): %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Create a map to structure the response as {"data": [...investigators]}
		response := map[string]interface{}{
			"data": investigadores,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response) // Encode the map
	}
}
