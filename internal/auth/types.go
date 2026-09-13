package auth

import "time"

type Usuario struct {
	ID       string
	GrupoID  string
	Nome     string
	Email    string
	Password string
	Role     string
	Ativo    bool
	// SenhaProvisoria: a senha atual foi definida por um administrador e precisa
	// ser trocada no primeiro acesso.
	SenhaProvisoria bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type RefreshToken struct {
	ID        string
	UsuarioID string
	Token     string
	ExpiresAt time.Time
	Revoked   bool
	CreatedAt time.Time
	// GrupoID é o grupo ativo quando a sessão foi criada. Sem ele o /auth/refresh
	// não teria como saber qual grupo o usuário multi-grupo selecionou — o refresh
	// token é opaco e não carrega claims.
	GrupoID string
	// Contexto em que a sessão foi aberta. Sem ele a renovação silenciosa
	// devolveria a pessoa a um contexto que ela não escolheu — e grupo_id NULL
	// não serve para deduzir, porque já significa duas coisas.
	Contexto Contexto
}

type GrupoInfo struct {
	ID         string `json:"id"`
	Nome       string `json:"nome"`
	Slug       string `json:"slug"`
	SchemaName string `json:"schema_name"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse cobre dois cenários:
// 1. Login direto (único grupo ou grupo padrão): access_token + refresh_token preenchidos.
// 2. Seleção pendente (múltiplos grupos): needs_select=true, pre_auth_token + grupos preenchidos.
type LoginResponse struct {
	// Cenário 1 — login completo
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in,omitempty"`

	// Contexto e GrupoID dizem onde a sessão nasceu. O cliente precisa dos dois
	// para montar menu e badge sem redecidir a regra por conta própria.
	Contexto Contexto `json:"contexto,omitempty"`
	GrupoID  string   `json:"grupo_id,omitempty"`
	// SenhaProvisoria: a tela precisa obrigar a troca antes de qualquer outra
	// coisa. A trava de verdade esta na claim do token — ver RequireAuth.
	SenhaProvisoria bool `json:"senha_provisoria,omitempty"`

	// Cenário 2 — seleção de contexto pendente
	NeedsSelect  bool        `json:"needs_select,omitempty"`
	PreAuthToken string      `json:"pre_auth_token,omitempty"`
	Grupos       []GrupoInfo `json:"grupos,omitempty"`
	// PodePlataforma diz se "Plataforma" entra na tela de escolha. Vem do
	// servidor para a lista de destinos não ser calculada também no cliente.
	PodePlataforma bool `json:"pode_plataforma,omitempty"`
}

// ContextosResponse é o corpo de GET /auth/contextos.
type ContextosResponse struct {
	PodePlataforma bool        `json:"pode_plataforma"`
	Grupos         []GrupoInfo `json:"grupos"`
}

type SelectGrupoRequest struct {
	PreAuthToken string `json:"pre_auth_token"`
	// Contexto vazio é lido como "grupo": é o único valor que um cliente antigo
	// poderia ter querido dizer, e nunca concede a plataforma por omissão.
	Contexto Contexto `json:"contexto"`
	GrupoID  string   `json:"grupo_id"`
}

type TrocaGrupoRequest struct {
	// Contexto vazio e lido como "grupo". Ver contextoOuGrupo no handler.
	Contexto Contexto `json:"contexto"`
	GrupoID  string   `json:"grupo_id"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type TrocaSenhaRequest struct {
	SenhaAtual string `json:"senha_atual"`
	SenhaNova  string `json:"senha_nova"`
}

type MeResponse struct {
	ID       string   `json:"id"`
	GrupoID  string   `json:"grupo_id"`
	Nome     string   `json:"nome"`
	Email    string   `json:"email"`
	Role     string   `json:"role"`
	Contexto Contexto `json:"contexto"`
	// SenhaProvisoria manda a tela obrigar a troca antes de qualquer outra
	// coisa. Vem também na claim do token, que é onde a regra é imposta.
	SenhaProvisoria bool `json:"senha_provisoria"`
}
