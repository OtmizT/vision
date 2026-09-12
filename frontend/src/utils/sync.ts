/**
 * Leitura do estado de sincronização de uma empresa.
 *
 * Puro e testado porque a duração aqui **não é um dado armazenado**: o worker
 * calcula e só escreve no log (internal/worker/worker.go), então a tela precisa
 * derivá-la de dois timestamps que podem faltar em qualquer combinação. Cada
 * combinação significa uma coisa diferente, e tratar todas como "—" esconderia
 * justamente o job travado, que é o que o admin precisa ver.
 */

export interface JanelaJob {
  /** Quando o job começou. Null = nunca chegou a iniciar. */
  iniciado?: string | null
  /** Quando terminou. Null com `iniciado` preenchido = ainda em andamento. */
  concluido?: string | null
}

export type EstadoDuracao = 'concluida' | 'em_andamento' | 'sem_inicio'

export interface Duracao {
  estado: EstadoDuracao
  /** Segundos decorridos. Null quando não há início de onde contar. */
  segundos: number | null
}

/**
 * Duração do job, e o que ela significa.
 *
 * `agora` é parâmetro, e não `Date.now()` lá dentro, para o teste poder fixar o
 * relógio — uma duração "em andamento" depende do instante da leitura.
 *
 * Timestamp inválido cai em `sem_inicio` em vez de produzir NaN: um NaN chega
 * na tela como "NaN s", que parece defeito da tela e não do dado.
 */
export function duracaoJob(j: JanelaJob, agora: number = Date.now()): Duracao {
  const ini = j.iniciado ? Date.parse(j.iniciado) : NaN
  if (Number.isNaN(ini)) return { estado: 'sem_inicio', segundos: null }

  const fim = j.concluido ? Date.parse(j.concluido) : NaN
  if (Number.isNaN(fim)) {
    return { estado: 'em_andamento', segundos: Math.max(0, Math.round((agora - ini) / 1000)) }
  }
  // Relógios podem discordar entre o banco e o cliente; duração negativa não
  // significa nada e viraria "-3 s" na tela.
  return { estado: 'concluida', segundos: Math.max(0, Math.round((fim - ini) / 1000)) }
}

/** Duração em texto curto: 45s, 12min, 2h13. */
export function fmtDuracao(d: Duracao): string {
  if (d.segundos === null) return '—'
  const s = d.segundos
  if (s < 60) return `${s}s`
  if (s < 3600) return `${Math.round(s / 60)}min`
  const h = Math.floor(s / 3600)
  const min = Math.round((s % 3600) / 60)
  return min === 0 ? `${h}h` : `${h}h${String(min).padStart(2, '0')}`
}

/**
 * Situação da empresa, para colorir a linha e ordenar a atenção.
 *
 * Ordem deliberada: um job travado aparece como 'travado' e não como 'rodando',
 * porque "rodando há 6 horas" é o disfarce mais comum de um processo morto.
 */
export type SituacaoSync =
  | 'nunca'      // nenhum job registrado
  | 'travado'    // rodando, mas sem sinal de vida
  | 'rodando'
  | 'erro'
  | 'ok'
  | 'pausado'    // agendamento desligado

export interface EstadoEmpresa {
  sync_ativo: boolean
  job_status?: string
  job_zumbi?: boolean
}

export function situacao(e: EstadoEmpresa): SituacaoSync {
  if (e.job_zumbi) return 'travado'
  if (e.job_status === 'rodando' || e.job_status === 'pendente') return 'rodando'
  if (e.job_status === 'erro') return 'erro'
  /*
   * "Nunca rodou" vem ANTES de "pausado", e a ordem custou uma verificação para
   * aparecer: empresa sem linha em sync_control chega com `sync_ativo: false`,
   * porque o COALESCE do SQL não distingue "desligado" de "nunca configurado".
   * Com pausado primeiro, uma empresa recém-cadastrada se anunciava como
   * desligada — o oposto da ação que ela pede, que é terminar de configurar.
   */
  if (!e.job_status) return 'nunca'
  // Pausado depois do erro: uma empresa desligada que falhou na última execução
  // ainda precisa mostrar a falha, senão o erro some ao pausar.
  if (!e.sync_ativo) return 'pausado'
  return 'ok'
}

/** Quanto mais alto, mais cedo a linha aparece. Erro e travado vêm primeiro. */
export function pesoSituacao(s: SituacaoSync): number {
  const pesos: Record<SituacaoSync, number> = {
    travado: 5, erro: 4, rodando: 3, nunca: 2, pausado: 1, ok: 0,
  }
  return pesos[s]
}
