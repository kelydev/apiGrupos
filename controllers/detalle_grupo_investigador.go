package controllers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/GoogleCloudPlatform/golang-samples/run/helloworld/models"
	"github.com/GoogleCloudPlatform/golang-samples/run/helloworld/repository"
	"github.com/gorilla/mux"
)

// CreateDetalleGrupoInvestigadorHandler handles creating a new relationship between a group and an investigator.
func CreateDetalleGrupoInvestigadorHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var detalle models.DetalleGrupoInvestigador
		if err := json.NewDecoder(r.Body).Decode(&detalle); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := repository.CreateDetalleGrupoInvestigador(db, &detalle); err != nil {
			log.Printf("Error creating group-investigator relationship: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(detalle)
	}
}

// GetDetalleGrupoInvestigadorHandler handles fetching a single relationship detail by its ID.
func GetDetalleGrupoInvestigadorHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		idStr := vars["id"]
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid detail ID", http.StatusBadRequest)
			return
		}

		detalle, err := repository.GetDetalleGrupoInvestigadorByID(db, id)
		if err != nil {
			log.Printf("Error getting detail by ID: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if detalle == nil {
			http.Error(w, "Detail not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(detalle)
	}
}

// UpdateDetalleGrupoInvestigadorHandler handles updating an existing relationship detail.
func UpdateDetalleGrupoInvestigadorHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		idStr := vars["id"]
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid detail ID", http.StatusBadRequest)
			return
		}

		var detalle models.DetalleGrupoInvestigador
		if err := json.NewDecoder(r.Body).Decode(&detalle); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Ensure the ID in the body matches the ID in the URL
		detalle.ID = id

		if err := repository.UpdateDetalleGrupoInvestigador(db, &detalle); err != nil {
			log.Printf("Error updating detail: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(detalle)
	}
}

// DeleteDetalleGrupoInvestigadorHandler handles deleting a specific relationship detail by its ID.
func DeleteDetalleGrupoInvestigadorHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		idStr := vars["id"]
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid detail ID", http.StatusBadRequest)
			return
		}

		if err := repository.DeleteDetalleGrupoInvestigador(db, id); err != nil {
			log.Printf("Error deleting detail: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// GetDetallesByGrupoHandler handles fetching all relationship details for a given group ID.
func GetDetallesByGrupoHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		grupoIDStr := vars["grupoID"]
		grupoID, err := strconv.Atoi(grupoIDStr)
		if err != nil {
			http.Error(w, "Invalid group ID", http.StatusBadRequest)
			return
		}

		detalles, err := repository.GetDetallesByGrupoID(db, grupoID)
		if err != nil {
			log.Printf("Error getting details by group ID: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(detalles)
	}
}
