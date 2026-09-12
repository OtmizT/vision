import api from './client'

export interface SyncJob {
  id:            string
  empresa_id:    string
  tipo:          string
  status:        'pendente' | 'rodando' | 'concluido' | 'erro'
  erro:          string
  iniciado_at:   string | null
  concluido_at:  string | null
  created_at:    string
}

export interface SyncControl {
  id:                        string
  empresa_id:                string
  ativo:                     boolean
  /**
   * Dois intervalos desde a migration 000012, que separou incremental de full.
   * Este tipo declarava `intervalo_min`, campo que deixou de existir ali — e o
   * SyncView redeclarava a interface certa localmente para contornar.
   */
  intervalo_incremental_min: number
  intervalo_full_dias:       number
  ultimo_sync_at:            string | null
  proximo_sync_at:           string | null
  ultimo_full_sync_at:       string | null
  proximo_full_sync_at:      string | null
}

/**
 * Uma empresa na visão geral de sync do admin global: agendamento e resultado
 * do último job, de todos os grupos.
 */
export interface EmpresaSync {
  grupo_id:     string
  grupo_nome:   string
  empresa_id:   string
  empresa_nome: string
  status:       string
  status_sync:  string

  sync_ativo:                 boolean
  intervalo_incremental_min?: number
  intervalo_full_dias?:       number
  ultimo_sync_at:             string | null
  proximo_sync_at:            string | null
  ultimo_full_sync_at:        string | null
  proximo_full_sync_at:       string | null

  /** Vazio quando a empresa nunca sincronizou — diferente de ter falhado. */
  job_id?:          string
  job_tipo?:        string
  job_status?:      string
  job_iniciado_at:  string | null
  job_concluido_at: string | null
  job_erro?:        string
  job_registros:    number
  /** Rodando sem sinal de vida. Ver PrazoHeartbeat em internal/sync/types.go. */
  job_zumbi:        boolean
}

export interface SyncStatus {
  empresa_id: string
  controle:   SyncControl | null
  ultimo_job: SyncJob | null
}

/**
 * Frescor dos dados do grupo. `ultimo_sync_at` vem null quando nenhuma empresa
 * do grupo concluiu sync — o backend devolve null em vez de uma data de época,
 * que apareceria como 1970 na interface.
 */
export interface UltimaAtualizacao {
  ultimo_sync_at: string | null
}

export const syncApi = {
  // O grupo sai das claims do token; não recebe parâmetro de propósito.
  ultimaAtualizacao: () =>
    api.get<{ data: UltimaAtualizacao }>('/sync/ultima-atualizacao'),

  status: (empresaId: string) =>
    api.get(`/sync/${empresaId}/status`),

  jobs: (empresaId: string, params?: { page?: number; per_page?: number }) =>
    api.get(`/sync/${empresaId}/jobs`, { params }),

  forcar: (empresaId: string, tipo: 'manual' | 'full') =>
    api.post(`/sync/${empresaId}/forcar`, { tipo }),

  // O corpo precisa casar com ConfigurarRequest (internal/sync/types.go): este
  // helper mandava `intervalo_min`, que o backend ignora desde a 000012. Como
  // nenhuma tela o chamava, o erro nunca apareceu — e esperava quem chamasse.
  configurar: (
    empresaId: string,
    payload: { ativo: boolean; intervalo_incremental_min: number; intervalo_full_dias: number },
  ) => api.put(`/sync/${empresaId}/configurar`, payload),
}

/**
 * Rotas de administração de sync, todas restritas ao admin global.
 *
 * Estavam espalhadas como `api.get` cru dentro do SyncControlCenter; reunidas
 * aqui, o contrato fica num lugar só e a tela para de conhecer URLs.
 */
export const syncAdminApi = {
  /** Todas as empresas de todos os grupos, com o estado de sync. */
  empresas: () => api.get<{ data: EmpresaSync[] }>('/admin/sync/empresas'),

  overview:   () => api.get('/admin/sync/overview'),
  jobsAtivos: () => api.get('/admin/sync/jobs/ativos'),
  dlq:        () => api.get('/admin/sync/dlq'),

  cancelarJob: (jobId: string)  => api.post(`/admin/sync/jobs/${jobId}/cancelar`),
  retryPagina: (pageId: string) => api.post(`/admin/sync/pages/${pageId}/retry`),
  startupRecovery: ()           => api.post('/admin/sync/startup-recovery'),
}
