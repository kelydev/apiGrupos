package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	// Importa el driver de PostgreSQL
	_ "github.com/lib/pq"
)

// InitDB initializes and returns a database connection.
func InitDB() (*sql.DB, error) {
	log.Print("initializing postgresql database connection...")

	// Usa los NOMBRES de las variables de entorno
	dbUser := os.Getenv("DB_USER")         // Nombre de la variable, ej: postgres
	dbPassword := os.Getenv("DB_PASSWORD") // Nombre de la variable, ej: 123456
	dbHost := os.Getenv("DB_HOST")         // Nombre de la variable, ej: localhost
	dbPort := os.Getenv("DB_PORT")         // Nombre de la variable, ej: 5432
	dbName := os.Getenv("DB_NAME")         // Nombre de la variable, ej: db_PIUnamba
	dbSSLMode := os.Getenv("DB_SSLMODE")   // Opcional, ej: disable

	// Validaciones básicas (opcional pero recomendado)
	if dbUser == "" || dbPassword == "" || dbHost == "" || dbPort == "" || dbName == "" {
		log.Fatal("Database environment variables DB_USER, DB_PASSWORD, DB_HOST, DB_PORT, DB_NAME must be set")
	}
	if dbSSLMode == "" {
		dbSSLMode = "disable" // Valor por defecto si no se especifica
	}

	// Construye el DSN (Data Source Name) para PostgreSQL
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)

	var err error
	// Usa "postgres" como nombre del driver
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	err = db.Ping()
	if err != nil {
		db.Close() // Cierra la conexión si el ping falla
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("PostgreSQL Database connection successfully established")
	return db, nil
}
