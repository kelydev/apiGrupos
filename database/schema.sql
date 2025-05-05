-- Table: usuario (Application Users)
CREATE TABLE Usuario (
    idUsuario SERIAL PRIMARY KEY,            -- Changed from id_usuario
    -- Removed supabase_user_id UUID UNIQUE NOT NULL,
    email VARCHAR(150) UNIQUE NOT NULL,
    password TEXT NOT NULL,                   -- Added password field (will store hash)
    -- Removed rol_aplicacion
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP, 
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Table: Investigador (Researchers)
CREATE TABLE Investigador (
    idInvestigador SERIAL PRIMARY KEY, -- SERIAL is PostgreSQL's auto-incrementing integer
    nombre VARCHAR(100) NOT NULL,
    apellido VARCHAR(100) NOT NULL,
    createdAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP -- Sets timestamp on creation only
);

-- Table: Grupo (Research Groups)
CREATE TABLE Grupo (
    idGrupo SERIAL PRIMARY KEY,
    nombre VARCHAR(150) NOT NULL,
    numeroResolucion VARCHAR(100) NOT NULL,
    lineaInvestigacion VARCHAR(200) NOT NULL,
    tipoInvestigacion VARCHAR(100) NOT NULL,
    fechaRegistro DATE NOT NULL,
    archivo VARCHAR(255), -- Assuming this stores a file path or name
    createdAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP -- Sets timestamp on creation only
);

-- Table: Grupo_Investigador (Associative table for Groups and Researchers)
CREATE TABLE Grupo_Investigador (
    idGrupo_Investigador SERIAL PRIMARY KEY,
    idGrupo INT NOT NULL,
    idInvestigador INT NOT NULL,
    rol VARCHAR(50) NOT NULL, -- e.g., 'Coordinador' or 'Integrante'
    createdAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP, -- Sets timestamp on creation only
    FOREIGN KEY (idGrupo) REFERENCES Grupo(idGrupo) ON DELETE CASCADE,
    FOREIGN KEY (idInvestigador) REFERENCES Investigador(idInvestigador) ON DELETE CASCADE
);

-- Función para actualizar updatedAt
CREATE OR REPLACE FUNCTION actualizar_updatedat()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP; -- Corrected column name
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers para cada tabla que necesita updatedAt

-- Usuario (Updated trigger to use new table name and function)
CREATE TRIGGER trigger_updatedat_usuario
BEFORE UPDATE ON Usuario
FOR EACH ROW
EXECUTE FUNCTION actualizar_updatedat();

-- Investigador
CREATE TRIGGER trigger_updatedat_investigador
BEFORE UPDATE ON Investigador
FOR EACH ROW
EXECUTE FUNCTION actualizar_updatedat();

-- Grupo
CREATE TRIGGER trigger_updatedat_grupo
BEFORE UPDATE ON Grupo
FOR EACH ROW
EXECUTE FUNCTION actualizar_updatedat();

-- Grupo_Investigador
CREATE TRIGGER trigger_updatedat_grupo_investigador
BEFORE UPDATE ON grupo_investigador
FOR EACH ROW
EXECUTE FUNCTION actualizar_updatedat();

CREATE EXTENSION IF NOT EXISTS unaccent;