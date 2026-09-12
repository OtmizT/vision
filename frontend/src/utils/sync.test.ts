import { describe, it, expect } from 'vitest'
import { duracaoJob, fmtDuracao, situacao, pesoSituacao, type SituacaoSync } from './sync'

// Relógio fixo: uma duração "em andamento" depende do instante da leitura, e um
// teste que use Date.now() passa hoje e falha amanhã.
const AGORA = Date.parse('2026-09-12T12:00:00Z')

describe('duracaoJob', () => {
  it('job concluído: diferença entre os dois timestamps', () => {
    const d = duracaoJob({ iniciado: '2026-09-12T11:58:00Z', concluido: '2026-09-12T11:58:45Z' }, AGORA)
    expect(d).toEqual({ estado: 'concluida', segundos: 45 })
  })

  // O caso que o admin precisa ver: iniciou e nunca terminou.
  it('job em andamento conta até agora', () => {
    const d = duracaoJob({ iniciado: '2026-09-12T11:30:00Z', concluido: null }, AGORA)
    expect(d).toEqual({ estado: 'em_andamento', segundos: 1800 })
  })

  it('job travado devolve uma duração enorme, e não um traço', () => {
    const d = duracaoJob({ iniciado: '2026-09-11T12:00:00Z', concluido: null }, AGORA)
    expect(d.estado).toBe('em_andamento')
    expect(d.segundos).toBe(86400)
  })

  it('sem início não há de onde contar', () => {
    expect(duracaoJob({ iniciado: null, concluido: null }, AGORA))
      .toEqual({ estado: 'sem_inicio', segundos: null })
  })

  // Empresa que nunca sincronizou chega com os dois campos ausentes.
  it('campos ausentes não quebram', () => {
    expect(duracaoJob({}, AGORA).estado).toBe('sem_inicio')
  })

  // NaN chegaria na tela como "NaN s" e pareceria defeito da tela.
  it('timestamp inválido vira sem_inicio, não NaN', () => {
    const d = duracaoJob({ iniciado: 'não é data', concluido: null }, AGORA)
    expect(d.estado).toBe('sem_inicio')
    expect(d.segundos).toBeNull()
  })

  it('conclusão inválida com início válido conta como em andamento', () => {
    const d = duracaoJob({ iniciado: '2026-09-12T11:59:00Z', concluido: 'lixo' }, AGORA)
    expect(d).toEqual({ estado: 'em_andamento', segundos: 60 })
  })

  // Relógios do banco e do cliente podem discordar; "-3s" não significa nada.
  it('duração negativa é aparada em zero', () => {
    const d = duracaoJob({ iniciado: '2026-09-12T12:00:10Z', concluido: '2026-09-12T12:00:00Z' }, AGORA)
    expect(d.segundos).toBe(0)
  })
})

describe('fmtDuracao', () => {
  const f = (s: number | null) => fmtDuracao({ estado: 'concluida', segundos: s })

  it('segundos, minutos e horas', () => {
    expect(f(0)).toBe('0s')
    expect(f(45)).toBe('45s')
    expect(f(59)).toBe('59s')
    expect(f(60)).toBe('1min')
    expect(f(720)).toBe('12min')
    expect(f(3600)).toBe('1h')
    expect(f(8000)).toBe('2h13')
  })

  it('hora cheia não mostra os minutos', () => {
    expect(f(7200)).toBe('2h')
  })

  it('minutos abaixo de dez ficam com dois dígitos, para alinhar na coluna', () => {
    expect(f(3900)).toBe('1h05')
  })

  it('sem duração é um traço', () => {
    expect(f(null)).toBe('—')
  })
})

describe('situacao', () => {
  it('sem job algum é "nunca"', () => {
    expect(situacao({ sync_ativo: true })).toBe('nunca')
  })

  it('concluído com agendamento ligado é "ok"', () => {
    expect(situacao({ sync_ativo: true, job_status: 'concluido' })).toBe('ok')
  })

  it('rodando e pendente contam como rodando', () => {
    expect(situacao({ sync_ativo: true, job_status: 'rodando' })).toBe('rodando')
    expect(situacao({ sync_ativo: true, job_status: 'pendente' })).toBe('rodando')
  })

  // "Rodando há seis horas" é o disfarce mais comum de um processo morto.
  it('zumbi é "travado", não "rodando"', () => {
    expect(situacao({ sync_ativo: true, job_status: 'rodando', job_zumbi: true })).toBe('travado')
  })

  it('erro aparece como erro', () => {
    expect(situacao({ sync_ativo: true, job_status: 'erro' })).toBe('erro')
  })

  /*
   * Empresa sem linha em sync_control chega com sync_ativo false, porque o
   * COALESCE do SQL nao distingue "desligado" de "nunca configurado". Sem esta
   * ordem, a empresa recem-cadastrada se anuncia como desligada — foi o que a
   * verificacao no navegador mostrou.
   */
  it('nunca rodou vence pausado quando nao ha job algum', () => {
    expect(situacao({ sync_ativo: false })).toBe('nunca')
  })

  it('agendamento desligado é "pausado"', () => {
    expect(situacao({ sync_ativo: false, job_status: 'concluido' })).toBe('pausado')
  })

  // Pausar não pode apagar a falha anterior da tela.
  it('empresa pausada que falhou continua mostrando o erro', () => {
    expect(situacao({ sync_ativo: false, job_status: 'erro' })).toBe('erro')
  })

  it('empresa pausada e travada continua travada', () => {
    expect(situacao({ sync_ativo: false, job_status: 'rodando', job_zumbi: true })).toBe('travado')
  })
})

describe('pesoSituacao', () => {
  it('ordena o que exige ação primeiro', () => {
    const ordem: SituacaoSync[] = ['ok', 'pausado', 'nunca', 'rodando', 'erro', 'travado']
    const pesos = ordem.map(pesoSituacao)
    expect([...pesos].sort((a, b) => a - b)).toEqual(pesos)
  })

  it('travado vem antes de erro', () => {
    expect(pesoSituacao('travado')).toBeGreaterThan(pesoSituacao('erro'))
  })
})
