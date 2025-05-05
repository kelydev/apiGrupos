# API de Grupos de Investigación

Backend para la gestión de grupos de investigación, desarrollado en Go.

## Guía de Instalación y Ejecución

Sigue estos pasos para clonar, configurar y ejecutar el proyecto en tu entorno local.

### 1. Prerrequisitos

Asegúrate de tener instalado lo siguiente:

*   **Go:** Versión 1.18 o superior (verifica con `go version`).
*   **Git:** Para clonar el repositorio (verifica con `git --version`).
*   **PostgreSQL:** Una instancia de base de datos PostgreSQL en ejecución. Puedes instalarla localmente o usar Docker.
*   **Variables de Entorno:** Un archivo `.env` para configurar las credenciales de la base de datos y el secreto JWT.

### 2. Clonar el Repositorio

Abre tu terminal y clona el proyecto usando Git:

```bash
git clone <URL_DEL_REPOSITORIO> # Reemplaza <URL_DEL_REPOSITORIO> con la URL real
cd apiGrupos # O el nombre del directorio clonado
```

### 3. Configurar Variables de Entorno

Este proyecto utiliza variables de entorno para la configuración sensible. Necesitarás crear un archivo `.env` en la raíz del proyecto.

1.  **Crea el archivo:**
    ```bash
    # Puedes copiar el archivo de ejemplo si existe
    # cp .env.example .env 
    # O crearlo manualmente
    touch .env 
    ```
2.  **Edita el archivo `.env`** y añade las siguientes variables con tus valores correspondientes:
    ```dotenv
    # PostgreSQL Database Configuration
    DB_USER=tu_usuario_postgres
    DB_PASSWORD=tu_contraseña_postgres
    DB_HOST=localhost # O la IP/host de tu servidor PostgreSQL
    DB_PORT=5432    # El puerto estándar de PostgreSQL
    DB_NAME=db_PIUnamba # O el nombre de tu base de datos
    DB_SSLMODE=disable # O 'require'/'verify-full' si usas SSL

    # JWT Secret Key (Usa una clave secreta segura y larga)
    JWT_SECRET=tu_super_secreto_jwt_muy_largo_y_seguro
    ```
    **¡Importante!** Asegúrate de que `JWT_SECRET` sea una cadena larga y aleatoria para mayor seguridad.

### 4. Dependencias del Proyecto

Este proyecto utiliza Go Modules para gestionar sus dependencias. El archivo `go.mod` en la raíz del proyecto define las bibliotecas externas necesarias. Las dependencias directas principales son:

*   `github.com/gorilla/mux`: Router HTTP.
*   `github.com/lib/pq`: Driver de PostgreSQL.
*   `github.com/joho/godotenv`: Carga de variables de entorno desde archivos `.env`.
*   `github.com/rs/cors`: Middleware para manejar CORS.
*   `github.com/golang-jwt/jwt/v5`: Para la generación y validación de tokens JWT.
*   `golang.org/x/crypto`: Utilizado para el hash de contraseñas (bcrypt).

**Instalación:**

Una vez dentro del directorio del proyecto, el siguiente comando descargará e instalará automáticamente todas las dependencias (directas e indirectas) listadas en `go.mod` y `go.sum`:

```bash
go mod tidy
# o alternativamente: go mod download
```

### 5. Configurar la Base de Datos

1.  **Accede a tu instancia de PostgreSQL** (usando `psql` o una herramienta gráfica como pgAdmin).
2.  **Crea la base de datos** si aún no existe (el nombre debe coincidir con `DB_NAME` en tu `.env`):
    ```sql
    CREATE DATABASE db_PIUnamba; -- O el nombre que hayas elegido
    ```
3.  **Conéctate a la base de datos recién creada.**
4.  **Ejecuta el script del esquema** para crear las tablas y funciones necesarias. El contenido del script se encuentra en `database/schema.sql`. Puedes copiar y pegar su contenido en tu cliente `psql` o ejecutarlo desde un archivo:
    ```bash
    psql -h tu_host -p tu_puerto -U tu_usuario -d tu_basedatos -f database/schema.sql
    ```
    (Reemplaza los placeholders con tus valores).

### 6. Ejecutar la Aplicación

Ahora puedes iniciar el servidor de la API:

```bash
go run main.go
```

Deberías ver un mensaje indicando que el servidor está escuchando en el puerto `3000` (o el puerto definido por la variable de entorno `PORT`):

```
INFO[...] starting server...
INFO[...] initializing postgresql database connection...
INFO[...] PostgreSQL Database connection successfully established
INFO[...] listening on port 3000
```

### 7. Probar la API

Puedes probar los endpoints usando herramientas como `curl`, Postman, Insomnia, o directamente desde tu frontend (asegúrate de que la configuración CORS en `main.go` permita el origen de tu frontend).

**Ejemplos:**

*   `GET http://localhost:3000/investigadores/all`
*   `GET http://localhost:3000/grupos?investigador=Ana%20Lopez`
*   `POST http://localhost:3000/register` (con un cuerpo JSON: `{"email":"test@example.com", "password":"tu_password"}`)

---

*Este README asume una configuración de desarrollo local. Para producción, considera pasos adicionales como compilación, contenedores (Docker), gestión de secretos más robusta y configuración de un servidor web/proxy inverso.*
