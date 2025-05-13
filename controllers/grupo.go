package controllers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

const (
	maxUploadSize = 10 * 1024 * 1024
	timeFormat    = "2006-01-02"
)

var (
	driveService  *drive.Service
	driveFolderID string
)

// init se ejecuta una vez al iniciar el paquete
func init() {
	// Cargar variables de entorno desde .env
	err := godotenv.Load() // Asume .env en el directorio de ejecución
	if err != nil {
		log.Println("Advertencia: No se pudo cargar el archivo .env, se intentará usar variables de entorno del sistema:", err)
	}

	credentialsPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	driveFolderID = os.Getenv("GOOGLE_DRIVE_FOLDER_ID")

	if credentialsPath == "" {
		log.Fatal("La variable de entorno GOOGLE_APPLICATION_CREDENTIALS no está configurada. Debe ser la ruta a su archivo JSON de credenciales.")
	}
	if driveFolderID == "" {
		log.Fatal("La variable de entorno GOOGLE_DRIVE_FOLDER_ID no está configurada.")
	}

	ctx := context.Background()

	// Leer el contenido del archivo de credenciales JSON
	credsBytes, err := os.ReadFile(credentialsPath)
	if err != nil {
		log.Fatalf("No se pudo leer el archivo de credenciales JSON desde la ruta especificada en GOOGLE_APPLICATION_CREDENTIALS (%s): %v", credentialsPath, err)
	}

	// Crear credenciales a partir del contenido del archivo JSON
	creds, err := google.CredentialsFromJSON(ctx, credsBytes, drive.DriveFileScope)
	if err != nil {
		log.Fatalf("No se pudieron crear las credenciales de Google a partir del archivo JSON. Asegúrese de que el archivo sea válido y contenga una clave privada PEM correcta: %v", err)
	}

	// Crear el cliente HTTP con las credenciales
	client := oauth2.NewClient(ctx, creds.TokenSource)

	// Crear el servicio de Drive
	driveService, err = drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("No se pudo crear el servicio de Drive: %v", err)
	}
	log.Println("Servicio de Google Drive inicializado correctamente.")
}

// constructDriveLink genera el enlace web de visualización para un ID de archivo de Drive
func constructDriveLink(fileID *string) *string {
	if fileID != nil && *fileID != "" {
		// Usar https://drive.google.com/file/d/FILE_ID/view como formato estándar
		link := fmt.Sprintf("https://drive.google.com/file/d/%s/view", *fileID)
		return &link
	}
	// Si no hay fileID, devuelve nil
	return nil
}

// Función auxiliar para crear oauth2.Config desde credenciales
func oauth2ConfigFromCredentials(creds *google.Credentials) *oauth2.Config {
	// Extraer ClientID y ClientSecret si están disponibles (típico para OAuth apps, menos para Service Accounts)
	// Para Service Accounts, el flujo es diferente y generalmente se usa JWTConfigFromJSON
	// Sin embargo, CredentialsFromJSON y el cliente resultante suelen manejar esto.
	// Si se usa un flujo OAuth de usuario, necesitarías el config.
	// Asumiendo credenciales de Service Account, el token source es suficiente.
	// Si necesitas un config explícito (p.ej., para obtener URL de autorización), tendrías que construirlo.
	// Para solo llamar APIs con Service Account, el client derivado de creds.TokenSource es suficiente.
	// Devolvemos nil o un config básico si es necesario en otros contextos. Aquí, el cliente directo basta.
	// Esta función podría necesitar ajustes dependiendo del TIPO EXACTO de credenciales (Service Account vs OAuth Client ID)
	// Para simplificar, asumimos que el client creado directamente es suficiente.
	return &oauth2.Config{
		ClientID:     creds.ProjectID, // O el ClientID específico si es app OAuth
		ClientSecret: "",              // No aplica directamente a Service Account para Config
		Endpoint:     google.Endpoint,
		Scopes:       []string{drive.DriveFileScope},
		// RedirectURL: "tu_redirect_url", // Si es app OAuth
	}
}

// Helper function to save uploaded file to Google Drive
func saveUploadedFile(r *http.Request, formKey string) (*string, error) {
	// Asegurarse de que el servicio de Drive esté inicializado
	if driveService == nil {
		return nil, fmt.Errorf("el servicio de Google Drive no está inicializado")
	}

	err := r.ParseMultipartForm(maxUploadSize)
	if err != nil {
		// Si no es multipart o falta el archivo, devolvemos nil, nil como antes
		if err == http.ErrNotMultipart || err == http.ErrMissingFile {
			log.Printf("Formulario no es multipart o falta archivo '%s'", formKey)
			return nil, nil // Indica que no se subió archivo, no es un error fatal aquí.
		}
		return nil, fmt.Errorf("error parsing multipart form: %w", err)
	}

	file, handler, err := r.FormFile(formKey)
	if err != nil {
		// Si el archivo específico no está, devolvemos nil, nil
		if err == http.ErrMissingFile {
			log.Printf("Campo de archivo '%s' no encontrado en el formulario", formKey)
			return nil, nil // Indica que no se subió archivo para este campo.
		}
		return nil, fmt.Errorf("error retrieving file '%s': %w", formKey, err)
	}
	defer file.Close()

	originalFilename := filepath.Base(handler.Filename)
	// Podríamos querer sanitizar el nombre aquí también si se usa en Drive
	uniqueFilename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), originalFilename)

	// Crear metadatos del archivo para Google Drive
	driveFile := &drive.File{
		Name:    uniqueFilename,
		Parents: []string{driveFolderID}, // ID de la carpeta donde guardar
	}

	// Subir el archivo
	createdFile, err := driveService.Files.Create(driveFile).Media(file).Do()
	if err != nil {
		// Intentar obtener más detalles del error si es posible
		googleErr, ok := err.(*googleapi.Error)
		if ok {
			log.Printf("Error detallado de Google API al subir archivo: Código=%d, Mensaje=%s, Errores=%v", googleErr.Code, googleErr.Message, googleErr.Errors)
		}
		return nil, fmt.Errorf("no se pudo crear el archivo en Google Drive: %w", err)
	}

	log.Printf("Archivo subido a Google Drive con ID: %s", createdFile.Id)
	// Devolver el ID del archivo de Drive en lugar de la ruta local
	return &createdFile.Id, nil
}

// removeFile elimina un archivo de Google Drive usando su ID
func removeFile(fileID *string) error {
	if fileID == nil || *fileID == "" {
		log.Println("No se proporcionó fileID para eliminar, omitiendo.")
		return nil // No hay nada que eliminar
	}
	// Asegurarse de que el servicio de Drive esté inicializado
	if driveService == nil {
		return fmt.Errorf("el servicio de Google Drive no está inicializado para eliminar archivo")
	}

	err := driveService.Files.Delete(*fileID).Do()
	if err != nil {
		// Podríamos querer verificar si el error es "not found" y tratarlo como éxito
		googleErr, ok := err.(*googleapi.Error)
		if ok && googleErr.Code == 404 {
			log.Printf("El archivo con ID '%s' no fue encontrado en Drive (quizás ya fue eliminado), considerando la operación exitosa.", *fileID)
			return nil // El archivo no existe, objetivo cumplido.
		}
		log.Printf("Error al eliminar archivo de Google Drive (ID: %s): %v", *fileID, err)
		return fmt.Errorf("error eliminando archivo '%s' de Google Drive: %w", *fileID, err)
	}

	log.Printf("Archivo con ID '%s' eliminado de Google Drive correctamente.", *fileID)
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

		// Construir enlaces para los archivos ANTES de enviar la respuesta
		for i := range gruposConDetalles {
			// Asumiendo que GrupoWithInvestigadores tiene un campo Grupo (models.Grupo) que contiene Archivo
			gruposConDetalles[i].Grupo.Archivo = constructDriveLink(gruposConDetalles[i].Grupo.Archivo)
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

		// Construir el enlace antes de enviar
		grupo.Archivo = constructDriveLink(grupo.Archivo)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(grupo)
	}
}

// CreateGrupoHandler handles creating a new group with potential file upload.
// Expects multipart/form-data
func CreateGrupoHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Llama a la nueva función saveUploadedFile que usa Drive
		fileID, err := saveUploadedFile(r, "archivo") // Ahora devuelve fileID o nil
		if err != nil {
			log.Printf("Error subiendo archivo a Drive durante creación de grupo: %v", err)
			// Distinguir errores de subida vs. errores de formulario
			if strings.Contains(err.Error(), "parsing multipart form") || strings.Contains(err.Error(), "request body too large") {
				http.Error(w, fmt.Sprintf("Error procesando formulario: %v", err), http.StatusBadRequest)
			} else if strings.Contains(err.Error(), "Google Drive") {
				// Error específico de Drive
				http.Error(w, "Error interno del servidor al subir archivo a Google Drive", http.StatusInternalServerError)
			} else {
				// Otro error inesperado durante saveUploadedFile
				http.Error(w, "Error interno del servidor procesando el archivo", http.StatusInternalServerError)
			}
			return // Detener ejecución si hubo error en saveUploadedFile
		}

		// fileID será nil si no se subió archivo o hubo error leve (no fatal) como ErrMissingFile
		// fileID tendrá el ID de Drive si la subida fue exitosa.

		var g models.Grupo
		g.Nombre = r.FormValue("nombre")
		g.NumeroResolucion = r.FormValue("numeroResolucion")
		g.LineaInvestigacion = r.FormValue("lineaInvestigacion")
		g.TipoInvestigacion = r.FormValue("tipoInvestigacion")

		fechaStr := r.FormValue("fechaRegistro")
		if fechaStr != "" {
			parsedDate, err := time.Parse(timeFormat, fechaStr)
			if err != nil {
				_ = removeFile(fileID) // Intentar eliminar el archivo de Drive si ya se subió
				http.Error(w, fmt.Sprintf("Formato inválido para fechaRegistro. Use %s", timeFormat), http.StatusBadRequest)
				return
			}
			g.FechaRegistro = parsedDate
		}

		if g.Nombre == "" || g.NumeroResolucion == "" || g.LineaInvestigacion == "" || g.TipoInvestigacion == "" {
			_ = removeFile(fileID) // Intentar eliminar el archivo de Drive si ya se subió
			http.Error(w, "Faltan campos de texto requeridos: nombre, numeroResolucion, lineaInvestigacion, tipoInvestigacion", http.StatusBadRequest)
			return
		}
		if g.FechaRegistro.IsZero() {
			_ = removeFile(fileID) // Intentar eliminar el archivo de Drive si ya se subió
			http.Error(w, fmt.Sprintf("Falta campo requerido o inválido: fechaRegistro (use formato %s)", timeFormat), http.StatusBadRequest)
			return
		}

		// Asignar el fileID (puede ser nil) al campo Archivo del grupo
		g.Archivo = fileID

		// Intentar crear el grupo en la BD
		if err := repository.CreateGrupo(db, &g); err != nil {
			log.Printf("Error creando grupo en repositorio: %v", err)
			_ = removeFile(fileID) // Si falla la BD, intentar eliminar el archivo de Drive
			http.Error(w, "Error interno del servidor guardando grupo", http.StatusInternalServerError)
			return
		}

		// Si todo fue bien:
		// Construir el enlace ANTES de enviar la respuesta
		g.Archivo = constructDriveLink(g.Archivo)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(g) // Devolver el grupo con el enlace (o nil)
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
			http.Error(w, "ID de grupo inválido", http.StatusBadRequest)
			return
		}

		// 1. Obtener el grupo existente para saber el ID del archivo antiguo (si existe)
		existingGrupo, err := repository.GetGrupoByID(db, id)
		if err != nil {
			log.Printf("Error obteniendo grupo por ID para actualizar: %v", err)
			http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
			return
		}
		if existingGrupo == nil {
			http.Error(w, "Grupo no encontrado para actualizar", http.StatusNotFound)
			return
		}
		oldFileID := existingGrupo.Archivo // Guardamos el ID del archivo antiguo (puede ser nil)

		// 2. Intentar subir un nuevo archivo (usando la función modificada)
		newFileID, err := saveUploadedFile(r, "archivo") // Devuelve el nuevo ID de Drive o nil
		if err != nil {
			log.Printf("Error subiendo archivo a Drive durante actualización de grupo: %v", err)
			// Manejar errores de subida como en CreateGrupoHandler
			if strings.Contains(err.Error(), "parsing multipart form") || strings.Contains(err.Error(), "request body too large") {
				http.Error(w, fmt.Sprintf("Error procesando formulario: %v", err), http.StatusBadRequest)
			} else if strings.Contains(err.Error(), "Google Drive") {
				http.Error(w, "Error interno del servidor al subir archivo a Google Drive", http.StatusInternalServerError)
			} else {
				http.Error(w, "Error interno del servidor procesando el archivo", http.StatusInternalServerError)
			}
			return // Detener si la subida falló
		}
		// newFileID es el ID del nuevo archivo si se subió, o nil si no se subió uno nuevo.

		// 3. Preparar los datos del grupo actualizado
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
				_ = removeFile(newFileID) // Si hubo error de fecha, eliminar el nuevo archivo si se subió
				http.Error(w, fmt.Sprintf("Formato inválido para fechaRegistro. Use %s", timeFormat), http.StatusBadRequest)
				return
			}
			updatedGrupo.FechaRegistro = parsedDate
		} else {
			// Mantener fecha existente si no se proporciona una nueva
			updatedGrupo.FechaRegistro = existingGrupo.FechaRegistro
		}

		// Mantener valores existentes si los campos del formulario están vacíos
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

		// 4. Determinar el ID del archivo final y si hay que borrar el antiguo
		var fileIDToDelete *string = nil
		if newFileID != nil {
			// Se subió un archivo nuevo. Usamos su ID.
			updatedGrupo.Archivo = newFileID
			// Si había un archivo antiguo diferente, marcarlo para borrar.
			if oldFileID != nil && *oldFileID != "" && *oldFileID != *newFileID {
				fileIDToDelete = oldFileID
			}
		} else {
			// No se subió un archivo nuevo, mantener el ID antiguo.
			updatedGrupo.Archivo = oldFileID
		}
		// Nota: No consideramos el caso de "eliminar" explícitamente un archivo existente sin reemplazarlo.
		// Si se quisiera eso, se necesitaría un campo adicional en el form, ej: "eliminarArchivo=true".

		// 5. Actualizar el grupo en la base de datos
		if err := repository.UpdateGrupo(db, &updatedGrupo); err != nil {
			log.Printf("Error actualizando grupo en repositorio: %v", err)
			// Si falla la BD, NO borrar el archivo antiguo, pero SÍ borrar el nuevo si se subió uno.
			_ = removeFile(newFileID)
			http.Error(w, "Error interno del servidor actualizando grupo", http.StatusInternalServerError)
			return
		}

		// 6. Si la actualización de la BD fue exitosa, borrar el archivo antiguo (si aplica)
		if fileIDToDelete != nil {
			err := removeFile(fileIDToDelete) // Usar la función modificada
			if err != nil {
				// Solo registrar advertencia, la actualización principal fue exitosa.
				log.Printf("Advertencia: Error eliminando archivo antiguo de Drive '%s' después de actualizar grupo: %v", *fileIDToDelete, err)
			}
		}

		// 7. Enviar respuesta exitosa
		// Construir el enlace ANTES de enviar la respuesta
		updatedGrupo.Archivo = constructDriveLink(updatedGrupo.Archivo)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(updatedGrupo) // Devolver el grupo actualizado con el enlace correcto
	}
}

// DeleteGrupoHandler handles deleting a group by ID.
func DeleteGrupoHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		idStr := vars["id"]
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "ID de grupo inválido", http.StatusBadRequest)
			return
		}

		// ANTES de eliminar el grupo de la BD, obtener su info para saber qué archivo borrar
		grupo, err := repository.GetGrupoByID(db, id)
		if err != nil {
			// Si no se puede obtener el grupo, podría no existir o haber otro error
			log.Printf("Error obteniendo grupo %d antes de eliminar: %v", id, err)
			// Decidir si continuar o no. Si el grupo no existe, DeleteGrupo probablemente falle igual.
			// Podríamos devolver un error aquí o dejar que DeleteGrupo maneje el not found.
			// Por seguridad, si no podemos obtener la info, no intentamos borrar archivo de Drive.
			// Dejemos que DeleteGrupo maneje la lógica de la BD.
		}

		// Intentar eliminar el grupo de la base de datos
		if err := repository.DeleteGrupo(db, id); err != nil {
			// Comprobar si el error es porque no se encontró el grupo
			// (Esta comprobación depende de cómo DeleteGrupo señale "not found")
			// if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "not found") {
			//	 http.Error(w, "Grupo no encontrado", http.StatusNotFound)
			//	 return
			// }
			// Si es otro error:
			log.Printf("Error eliminando grupo %d de la BD: %v", id, err)
			http.Error(w, "Error interno del servidor al eliminar grupo", http.StatusInternalServerError)
			return
		}

		// Si la eliminación de la BD fue exitosa Y pudimos obtener la info del grupo antes:
		if grupo != nil && grupo.Archivo != nil && *grupo.Archivo != "" {
			log.Printf("Grupo %d eliminado de la BD, intentando eliminar archivo de Drive con ID: %s", id, *grupo.Archivo)
			err := removeFile(grupo.Archivo) // Usar la función modificada
			if err != nil {
				// Solo registrar advertencia, la eliminación del grupo fue exitosa.
				log.Printf("Advertencia: Error eliminando archivo de Drive '%s' después de eliminar grupo %d: %v", *grupo.Archivo, id, err)
			}
		} else if grupo != nil {
			log.Printf("Grupo %d eliminado de la BD, no tenía archivo asociado en Drive.", id)
		} else {
			log.Printf("Grupo %d eliminado de la BD, no se pudo obtener info previa para eliminar archivo de Drive asociado.", id)
		}

		w.WriteHeader(http.StatusNoContent) // Éxito
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

		// Construir el enlace antes de enviar
		if grupoWithInvestigadores != nil {
			// Asumiendo que GrupoWithInvestigadores tiene un campo Grupo (models.Grupo) que contiene Archivo
			grupoWithInvestigadores.Grupo.Archivo = constructDriveLink(grupoWithInvestigadores.Grupo.Archivo)
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
// **NOTA:** Este handler usa JSON, no multipart/form-data.
// La subida de archivos debería hacerse ANTES con CreateGrupoHandler
// y luego pasar el ID del archivo (o nil) en requestBody.Grupo.Archivo.
// La lógica actual de este handler NO interactúa con saveUploadedFile.
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
		grupoToCreate := requestBody.Grupo // Ya debería incluir el ID de Drive si se subió antes
		// Use lowercase snake_case names and $n placeholders
		groupInsertQuery := `INSERT INTO grupo (nombre, numeroResolucion, lineaInvestigacion, tipoInvestigacion, fechaRegistro, archivo) VALUES ($1, $2, $3, $4, $5, $6) RETURNING idGrupo`
		var grupoID int64 // Use int64 for Scan with RETURNING

		// Asegurarse de pasar nil si Archivo es nil o el valor si existe
		var archivoID interface{}
		if grupoToCreate.Archivo != nil {
			archivoID = *grupoToCreate.Archivo
		} else {
			archivoID = nil
		}

		err = tx.QueryRow(groupInsertQuery, grupoToCreate.Nombre, grupoToCreate.NumeroResolucion, grupoToCreate.LineaInvestigacion, grupoToCreate.TipoInvestigacion, grupoToCreate.FechaRegistro, archivoID).Scan(&grupoID)
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
		// Construir el enlace ANTES de enviar la respuesta
		grupoToCreate.Archivo = constructDriveLink(grupoToCreate.Archivo)
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

		// Enriquecer la respuesta para incluir los integrantes con su rol Y CONSTRUIR ENLACES
		var respuesta []map[string]interface{}
		for _, grupoConInt := range gruposConIntegrantes {
			// Asumiendo que 'grupoConInt["grupo"]' es un tipo que tiene un campo 'Archivo'
			// Necesitamos hacer type assertion y modificar el campo.
			if grupoData, ok := grupoConInt["grupo"].(models.Grupo); ok { // Ajusta models.Grupo si es otro tipo
				grupoData.Archivo = constructDriveLink(grupoData.Archivo)
				grupoConInt["grupo"] = grupoData // Reasignar el grupo modificado al mapa
			} else if grupoDataPtr, ok := grupoConInt["grupo"].(*models.Grupo); ok && grupoDataPtr != nil { // Caso puntero
				grupoDataPtr.Archivo = constructDriveLink(grupoDataPtr.Archivo)
				// No es necesario reasignar porque modificamos el puntero
			} else {
				// Manejar el caso en que la aserción falle o el tipo sea inesperado
				log.Printf("Advertencia: No se pudo convertir grupo a tipo esperado para construir enlace en GetGruposByInvestigadorHandler: %T", grupoConInt["grupo"])
			}

			respuesta = append(respuesta, map[string]interface{}{
				"grupo":       grupoConInt["grupo"], // Ya tiene el enlace construido
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

		// Construir enlaces para los archivos ANTES de enviar la respuesta
		for i := range gruposConDetalles {
			// Asumiendo que GrupoWithInvestigadores tiene un campo Grupo (models.Grupo) que contiene Archivo
			gruposConDetalles[i].Grupo.Archivo = constructDriveLink(gruposConDetalles[i].Grupo.Archivo)
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

// GetAllDetallesGrupoInvestigadorHandler retrieves all group-investigator relationships with pagination.
func GetAllDetallesGrupoInvestigadorHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Read pagination params
		page, limit := utils.GetPaginationParams(r)
		offset := (page - 1) * limit

		// Call the repository function to get all details
		detalles, totalItems, err := repository.GetAllDetallesGrupoInvestigador(db, limit, offset)
		if err != nil {
			log.Printf("Error getting all group-investigator details: %v", err)
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
			Data:       detalles,
			Pagination: pagination,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
