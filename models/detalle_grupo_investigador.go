package models

import "time"

// DetalleGrupoInvestigador represents the relationship between a group and an investigator.
type DetalleGrupoInvestigador struct {
	ID             int       `json:"idGrupoInvestigador" db:"id_grupo_investigador"`
	IDGrupo        int       `json:"idGrupo" db:"idGrupo"`
	IDInvestigador int       `json:"idInvestigador" db:"idInvestigador"`
	Rol            string    `json:"rol" db:"rol"`
	CreatedAt      time.Time `json:"createdAt" db:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt" db:"updatedAt"`
}
