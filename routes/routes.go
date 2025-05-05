package routes

import (
	"database/sql"
	"net/http"

	"github.com/GoogleCloudPlatform/golang-samples/run/helloworld/controllers"
	"github.com/GoogleCloudPlatform/golang-samples/run/helloworld/middleware"
	"github.com/gorilla/mux"
)

// SetupRoutes configures the application routes.
func SetupRoutes(db *sql.DB) *mux.Router {
	r := mux.NewRouter()

	// --- Authentication Routes (Public) ---
	r.HandleFunc("/register", controllers.RegisterHandler(db)).Methods("POST")
	r.HandleFunc("/login", controllers.LoginHandler(db)).Methods("POST")

	// --- Public GET Routes (No Auth Required) ---
	r.HandleFunc("/investigadores", controllers.GetInvestigadoresHandler(db)).Methods("GET")
	r.HandleFunc("/investigadores/all", controllers.GetAllInvestigadoresNoPaginationHandler(db)).Methods("GET")
	r.HandleFunc("/investigadores/{id}", controllers.GetInvestigadorHandler(db)).Methods("GET")
	r.HandleFunc("/investigadores/{idInvestigador}/grupos", controllers.GetGruposByInvestigadorHandler(db)).Methods("GET")
	r.HandleFunc("/grupos", controllers.GetGruposHandler(db)).Methods("GET")
	r.HandleFunc("/grupos/{id}", controllers.GetGrupoHandler(db)).Methods("GET")
	r.HandleFunc("/grupos/{id}/details", controllers.GetGrupoDetailsHandler(db)).Methods("GET")
	r.HandleFunc("/grupos/with-details", controllers.GetAllGruposWithDetailsHandler(db)).Methods("GET")
	r.HandleFunc("/detalles/{id}", controllers.GetDetalleGrupoInvestigadorHandler(db)).Methods("GET")
	r.HandleFunc("/grupos/{grupoID}/detalles", controllers.GetDetallesByGrupoHandler(db)).Methods("GET")

	// Static file server (public)
	fs := http.FileServer(http.Dir("./uploads/"))
	r.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", fs))

	// --- Protected Routes (Auth Required) ---

	// Create a subrouter for authenticated routes
	authRouter := r.PathPrefix("").Subrouter()
	authRouter.Use(middleware.JWTMiddleware) // Apply JWT middleware to this subrouter

	// Investigador (Create, Update, Delete)
	authRouter.HandleFunc("/investigadores", controllers.CreateInvestigadorHandler(db)).Methods("POST")
	authRouter.HandleFunc("/investigadores/{id}", controllers.UpdateInvestigadorHandler(db)).Methods("PUT")
	authRouter.HandleFunc("/investigadores/{id}", controllers.DeleteInvestigadorHandler(db)).Methods("DELETE")

	// Grupo (Create, Update, Delete, Create with Details)
	authRouter.HandleFunc("/grupos", controllers.CreateGrupoHandler(db)).Methods("POST") // Handles file upload
	authRouter.HandleFunc("/grupos/with-details", controllers.CreateGrupoWithDetailsHandler(db)).Methods("POST")
	authRouter.HandleFunc("/grupos/{id}", controllers.UpdateGrupoHandler(db)).Methods("PUT") // Handles file upload
	authRouter.HandleFunc("/grupos/{id}", controllers.DeleteGrupoHandler(db)).Methods("DELETE")

	// DetalleGrupoInvestigador (Create, Update, Delete)
	authRouter.HandleFunc("/detalles", controllers.CreateDetalleGrupoInvestigadorHandler(db)).Methods("POST")
	authRouter.HandleFunc("/detalles/{id}", controllers.UpdateDetalleGrupoInvestigadorHandler(db)).Methods("PUT")
	authRouter.HandleFunc("/detalles/{id}", controllers.DeleteDetalleGrupoInvestigadorHandler(db)).Methods("DELETE")

	return r
}
