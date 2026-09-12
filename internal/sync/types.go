package sync

import "time"

type SyncJob struct {
	ID          string     `json:"id"`
	EmpresaID   string     `json:"empresa_id"`
	Tipo        string     `json:"tipo"`
	Status      string     `json:"status"`
	Erro        string     `json:"erro"`
	IniciadoAt  *time.Time `json:"iniciado_at,omitempty"`
	ConcluidoAt *time.Time `json:"concluido_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	Executor    string     `json:"executor,omitempty"`
}

type SyncControl struct {
	ID                      string     `json:"id"`
	EmpresaID               string     `json:"empresa_id"`
	Ativo                   bool       `json:"ativo"`
	IntervaloIncrementalMin int        `json:"intervalo_incremental_min"`
	IntervaloFullDias       int        `json:"intervalo_full_dias"`
	UltimoSyncAt            *time.Time `json:"ultimo_sync_at,omitempty"`
	ProximoSyncAt           *time.Time `json:"proximo_sync_at,omitempty"`
	UltimoFullSyncAt        *time.Time `json:"ultimo_full_sync_at,omitempty"`
	ProximoFullSyncAt       *time.Time `json:"proximo_full_sync_at,omitempty"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
}

type StatusResponse struct {
	EmpresaID string       `json:"empresa_id"`
	Control   *SyncControl `json:"controle"`
	UltimoJob *SyncJob     `json:"ultimo_job,omitempty"`
}

type ForcarSyncRequest struct {
	Tipo     string `json:"tipo"`
	Executor string `json:"executor,omitempty"`
}

type ConfigurarRequest struct {
	Ativo                   bool `json:"ativo"`
	IntervaloIncrementalMin int  `json:"intervalo_incremental_min"`
	IntervaloFullDias       int  `json:"intervalo_full_dias"`
}

type EmpresaExecutorConfig struct {
	Executor  string    `json:"executor"`
	Ativo     bool      `json:"ativo"`
	Notas     *string   `json:"notas,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
	UpdatedBy *string   `json:"updated_by,omitempty"`
}

type UpdateExecutorConfigRequest struct {
	Ativo bool    `json:"ativo"`
	Notas *string `json:"notas,omitempty"`
}

type SyncJobProgress struct {
	Executor       string     `json:"executor"`
	Status         string     `json:"status"`
	PaginaAtual    *int       `json:"pagina_atual,omitempty"`
	TotalPaginas   *int       `json:"total_paginas,omitempty"`
	RegistrosProc  int        `json:"registros_proc"`
	RegistrosTotal *int       `json:"registros_total,omitempty"`
	Erro           *string    `json:"erro,omitempty"`
	IniciadoAt     *time.Time `json:"iniciado_at,omitempty"`
	ConcluidoAt    *time.Time `json:"concluido_at,omitempty"`
	UpdatedAt      time.Time  `json:"updated_at"`

	// Novos campos para inspeção de payload
	UltimoPayload  any     `json:"ultimo_payload,omitempty"`
	UltimoResponse any     `json:"ultimo_response,omitempty"`
	ErroPayload    any     `json:"erro_payload,omitempty"`
	ErroResponse   *string `json:"erro_response,omitempty"`
}

type ListParams struct {
	EmpresaID string
	Page      int
	PerPage   int
}

type JobStatusCount struct {
	Status string `json:"status"`
	Total  int64  `json:"total"`
}

type JobAtivoRow struct {
	ID                string     `json:"id"`
	EmpresaID         string     `json:"empresa_id"`
	EmpresaNome       string     `json:"empresa_nome"`
	GrupoNome         string     `json:"grupo_nome"`
	Tipo              string     `json:"tipo"`
	Status            string     `json:"status"`
	IniciadoAt        time.Time  `json:"iniciado_at"`
	UltimoHeartbeatAt *time.Time `json:"ultimo_heartbeat_at,omitempty"`
	IsZumbi           bool       `json:"is_zumbi"`
}

/*
PrazoHeartbeat e o silencio a partir do qual um job "rodando" e considerado
zumbi: o processo morreu sem marcar o job como concluido ou com erro.

ATENCAO: os mesmos 10 minutos estao escritos como INTERVAL na query
GetJobsAtivos (db/queries/sync.sql) e eram recalculados no cliente, em
SyncControlCenter.vue. Esta constante e a terceira copia — e a unica testavel.
Mexer no prazo exige mexer nas duas. Unificar exige parametrizar aquela query,
o que muda a assinatura dela e dos chamadores; ficou fora desta entrega.
*/
const PrazoHeartbeat = 10 * time.Minute

// EmpresaSyncRow e uma linha da visao geral de sync: uma empresa, com o estado
// do seu agendamento e o resultado do ultimo job.
//
// Reune tres tabelas que ate agora so eram lidas por grupo (empresas,
// sync_control, sync_jobs). A tela do admin global precisava de todas as
// empresas de uma vez, e montar isso no cliente custaria uma chamada por grupo.
type EmpresaSyncRow struct {
	GrupoID     string `json:"grupo_id"`
	GrupoNome   string `json:"grupo_nome"`
	EmpresaID   string `json:"empresa_id"`
	EmpresaNome string `json:"empresa_nome"`
	Status      string `json:"status"`
	StatusSync  string `json:"status_sync"`

	SyncAtivo               bool       `json:"sync_ativo"`
	IntervaloIncrementalMin *int32     `json:"intervalo_incremental_min,omitempty"`
	IntervaloFullDias       *int32     `json:"intervalo_full_dias,omitempty"`
	UltimoSyncAt            *time.Time `json:"ultimo_sync_at,omitempty"`
	ProximoSyncAt           *time.Time `json:"proximo_sync_at,omitempty"`
	UltimoFullSyncAt        *time.Time `json:"ultimo_full_sync_at,omitempty"`
	ProximoFullSyncAt       *time.Time `json:"proximo_full_sync_at,omitempty"`

	// Ultimo job. Vazio quando a empresa nunca sincronizou — e a tela precisa
	// distinguir "nunca rodou" de "rodou e falhou".
	JobID          string     `json:"job_id,omitempty"`
	JobTipo        string     `json:"job_tipo,omitempty"`
	JobStatus      string     `json:"job_status,omitempty"`
	JobIniciadoAt  *time.Time `json:"job_iniciado_at,omitempty"`
	JobConcluidoAt *time.Time `json:"job_concluido_at,omitempty"`
	JobErro        *string    `json:"job_erro,omitempty"`
	JobRegistros   int64      `json:"job_registros"`

	// Zumbi: rodando, mas sem heartbeat ha mais de 10 minutos. Mesmo criterio
	// de GetJobsAtivos, calculado aqui para nao depender de outra consulta.
	JobZumbi bool `json:"job_zumbi"`
}

type JobPage struct {
	ID             string     `json:"id"`
	JobID          string     `json:"job_id"`
	Modulo         string     `json:"modulo"`
	Pagina         int        `json:"pagina"`
	TotalPaginas   int        `json:"total_paginas"`
	Tentativas     int        `json:"tentativas"`
	MaxTentativas  int        `json:"max_tentativas"`
	ProximoRetryAt *time.Time `json:"proximo_retry_at,omitempty"`
}

type DLQPageRow struct {
	ID            string     `json:"id"`
	JobID         string     `json:"job_id"`
	EmpresaNome   string     `json:"empresa_nome"`
	GrupoNome     string     `json:"grupo_nome"`
	Modulo        string     `json:"modulo"`
	Pagina        int        `json:"pagina"`
	TotalPaginas  int        `json:"total_paginas"`
	Tentativas    int        `json:"tentativas"`
	MaxTentativas int        `json:"max_tentativas"`
	Erro          *string    `json:"erro,omitempty"`
	ConcluidoAt   *time.Time `json:"concluido_at,omitempty"`
}

type PageRow struct {
	ID                string     `json:"id"`
	Modulo            string     `json:"modulo"`
	Pagina            int        `json:"pagina"`
	TotalPaginas      int        `json:"total_paginas"`
	Status            string     `json:"status"`
	Tentativas        int        `json:"tentativas"`
	MaxTentativas     int        `json:"max_tentativas"`
	RegistrosGravados int        `json:"registros_gravados"`
	Erro              *string    `json:"erro,omitempty"`
	ProximoRetryAt    *time.Time `json:"proximo_retry_at,omitempty"`
	IniciadoAt        *time.Time `json:"iniciado_at,omitempty"`
	ConcluidoAt       *time.Time `json:"concluido_at,omitempty"`
}

// UltimaAtualizacao é o frescor dos dados do grupo. Ponteiro porque grupo sem
// nenhum sync concluído devolve null, e não uma data de época que o frontend
// exibiria como 1970.
type UltimaAtualizacao struct {
	UltimoSyncAt *time.Time `json:"ultimo_sync_at"`
}
