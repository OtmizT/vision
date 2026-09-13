package usuarios

import "time"

type Usuario struct {
	ID        string    `json:"id"`
	GrupoID   string    `json:"grupo_id"`
	Nome      string    `json:"nome"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Ativo     bool      `json:"ativo"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateRequest struct {
	Nome     string `json:"nome"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type CreateResult struct {
	Usuario      *Usuario `json:"usuario"`
	AddedToGroup bool     `json:"added_to_group"`
}

type UpdateRequest struct {
	Nome  string `json:"nome"`
	Role  string `json:"role"`
	Ativo bool   `json:"ativo"`
}

type UpdatePasswordRequest struct {
	Password string `json:"password"`
}

type ListParams struct {
	GrupoID string
	Page    int
	PerPage int
}

/*
Papeis que estas rotas podem gravar.

admin_global saiu da lista, e a ausencia dele e o ponto.

Estas rotas vivem sob /admin/grupos/{grupoID}/usuarios e sao operadas por
admin_grupo. Com admin_global aceito, um admin do cliente promovia alguem --
inclusive a si mesmo -- a administrador da plataforma inteira: o PUT grava
usuarios.role, que e a coluna global, e o CASE WHEN de GetRoleNoGrupo faz
admin_global vencer qualquer papel de grupo no proximo login.

Promover a plataforma deixa de ser possivel pela interface, que e exatamente a
regra de "existe um admin global". O banco reforca o mesmo pela migration
000030.
*/
var rolesValidas = map[string]bool{
	"admin_grupo": true,
	"viewer":      true,
}
