import api from './client'
import type { VisaoSpec } from '@/utils/visaospec'

export interface FonteResposta {
  ferramenta: string
  filtros: Record<string, unknown>
}

export interface MensagemChat {
  id?: string
  papel: 'usuario' | 'assistente'
  conteudo: string
  grafico?: VisaoSpec
  fonte?: FonteResposta
  criado_em?: string
  /** Só no cliente: marca a bolha que falhou, para permitir tentar de novo. */
  erro?: boolean
}

export interface RespostaChat {
  tipo: 'resposta' | 'grafico' | 'recusa'
  texto: string
  grafico?: VisaoSpec
  fonte?: FonteResposta
  tokens: number
}

/** Filtros da tela. Viram o padrão das ferramentas — é o que faz "e no mês
 *  passado?" ter sentido sem a pessoa repetir o período. */
export interface ContextoTela {
  ano?: number
  mes?: number
  aba?: string
}

export interface ConfigIA {
  provedor: string
  modelo: string
  base_url: string
  api_key_mascarada: string
  tem_chave: boolean
  max_tokens: number
  teto_tokens_dia: number
  ativo: boolean
  updated_at: string
  updated_by_email?: string
}

export interface GrupoIA {
  grupo_id: string
  grupo_nome: string
  grupo_slug: string
  ativa: boolean
  updated_at?: string
  updated_by_email?: string
}

export const iaApi = {
  disponivel: () => api.get<{ data: { disponivel: boolean } }>('/ia/disponivel'),

  perguntar: (pergunta: string, contexto: ContextoTela) =>
    api.post<{ data: RespostaChat }>('/ia/chat', { pergunta, contexto }),

  conversa: () => api.get<{ data: MensagemChat[] }>('/ia/conversa'),
  limpar: () => api.delete('/ia/conversa'),
}

export const iaConfigApi = {
  get: () => api.get<{ data: ConfigIA }>('/admin/ia-config'),

  /** api_key vazia PRESERVA a atual — ver internal/ia_config/types.go. */
  salvar: (payload: Partial<ConfigIA> & { api_key?: string; limpar_chave?: boolean }) =>
    api.put<{ data: ConfigIA }>('/admin/ia-config', payload),

  grupos: () => api.get<{ data: GrupoIA[] }>('/admin/ia-config/grupos'),
  setGrupo: (grupoID: string, ativa: boolean) =>
    api.put(`/admin/ia-config/grupos/${grupoID}`, { ativa }),
}
