package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/GoogleCloudPlatform/golang-samples/run/helloworld/database"
	"github.com/GoogleCloudPlatform/golang-samples/run/helloworld/routes" // Usa gorilla/mux
	"github.com/joho/godotenv"                                            // Para cargar variables de entorno desde .env
	"github.com/rs/cors"                                                  // Importar CORS para gorilla/mux
	// Se eliminan imports de gin
)

var db *sql.DB

// Se elimina struct Grupo si no se usa aquí

func main() {
	log.Print("starting server...")

	// Cargar variables de entorno desde .env
	err := godotenv.Load()
	if err != nil && !os.IsNotExist(err) {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	// Initialize database connection
	db, err = database.InitDB()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Setup routes using the routes package (gorilla/mux)
	r := routes.SetupRoutes(db)

	// --- Configuración de CORS usando rs/cors ---
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:4200"},                   // Origen permitido
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}, // Métodos permitidos
		AllowedHeaders:   []string{"Content-Type", "Authorization"},           // Cabeceras permitidas
		AllowCredentials: true,
		// Debug:            true, // Habilita logs de CORS si necesitas depurar
	})

	// Envolver el router 'r' con el handler CORS
	httpHandler := c.Handler(r)

	// Determine port for HTTP service.
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
		log.Printf("defaulting to port %s", port)
	}

	// Start HTTP server using net/http with the CORS handler
	log.Printf("listening on port %s", port)
	if err := http.ListenAndServe(":"+port, httpHandler); err != nil {
		log.Fatal(err)
	}
}
