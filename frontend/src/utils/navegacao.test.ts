import { describe, it, expect } from 'vitest'
import {
  rotaInicial, temDashboard, temSyncControl, destinoAposEntrar,
  contextoOuGrupo, rotuloContexto,
  ROTA_PLATAFORMA, ROTA_PADRAO, type Contexto,
} from './navegacao'

const CONTEXTOS: Contexto[] = ['plataforma', 'grupo']

describe('contextoOuGrupo', () => {
  it('reconhece os dois contextos', () => {
    expect(contextoOuGrupo('plataforma')).toBe('plataforma')
    expect(contextoOuGrupo('grupo')).toBe('grupo')
  })

  /*
   * O mesmo padrão do servidor: omissão nunca concede a plataforma. Aparece de
   * verdade entre o login e o primeiro /auth/me, quando `user` ainda é null.
   */
  it('ausente ou desconhecido nunca vira plataforma', () => {
    expect(contextoOuGrupo(undefined)).toBe('grupo')
    expect(contextoOuGrupo(null)).toBe('grupo')
    expect(contextoOuGrupo('')).toBe('grupo')
    expect(contextoOuGrupo('admin_global')).toBe('grupo')
  })
})

describe('rotaInicial', () => {
  it('plataforma vai para o Sync Control', () => {
    expect(rotaInicial('plataforma')).toBe(ROTA_PLATAFORMA)
  })

  it('grupo vai para o Dashboard', () => {
    expect(rotaInicial('grupo')).toBe(ROTA_PADRAO)
  })

  it('contexto ausente cai no padrão, não em undefined', () => {
    expect(rotaInicial(undefined)).toBe(ROTA_PADRAO)
    expect(rotaInicial(null)).toBe(ROTA_PADRAO)
  })

  it('todo contexto tem destino, e o destino é um caminho interno', () => {
    for (const c of CONTEXTOS) expect(rotaInicial(c).startsWith('/')).toBe(true)
  })
})

describe('temDashboard e temSyncControl', () => {
  it('cada contexto tem uma das duas telas, nunca as duas nem nenhuma', () => {
    for (const c of CONTEXTOS) {
      expect(temDashboard(c)).toBe(!temSyncControl(c))
    }
  })

  it('plataforma não tem Dashboard; grupo não tem Sync Control', () => {
    expect(temDashboard('plataforma')).toBe(false)
    expect(temSyncControl('plataforma')).toBe(true)
    expect(temDashboard('grupo')).toBe(true)
    expect(temSyncControl('grupo')).toBe(false)
  })

  /*
   * Menu e roteador precisam concordar: discordando, ou o item some e a URL
   * continua aberta, ou o item aparece e leva a um 403.
   */
  it('a rota inicial de um contexto é sempre uma tela que ele tem', () => {
    for (const c of CONTEXTOS) {
      const inicial = rotaInicial(c)
      if (inicial === ROTA_PADRAO) expect(temDashboard(c)).toBe(true)
      if (inicial === ROTA_PLATAFORMA) expect(temSyncControl(c)).toBe(true)
    }
  })
})

describe('rotuloContexto', () => {
  // Sem badge, o admin global dentro de um grupo se lê como "Admin Grupo" na
  // tela, sem pista de que é a mesma conta que administra a plataforma.
  it('plataforma se anuncia como Plataforma', () => {
    expect(rotuloContexto('plataforma')).toBe('Plataforma')
    expect(rotuloContexto('plataforma', 'Grupo Alpha')).toBe('Plataforma')
  })

  it('grupo usa o nome do grupo', () => {
    expect(rotuloContexto('grupo', 'Grupo Alpha')).toBe('Grupo Alpha')
  })

  // Um badge vazio é pior que um genérico: some da tela sem avisar.
  it('grupo sem nome ainda tem rótulo', () => {
    expect(rotuloContexto('grupo')).toBe('Grupo')
    expect(rotuloContexto('grupo', '   ')).toBe('Grupo')
  })
})

describe('destinoAposEntrar', () => {
  it('sem redirect, usa a rota do contexto', () => {
    expect(destinoAposEntrar('plataforma')).toBe(ROTA_PLATAFORMA)
    expect(destinoAposEntrar('grupo')).toBe(ROTA_PADRAO)
  })

  it('honra um redirect interno', () => {
    expect(destinoAposEntrar('grupo', '/usuarios')).toBe('/usuarios')
  })

  // Um link com redirect externo levaria o usuário recém-autenticado para fora.
  it('recusa redirect para outro site', () => {
    expect(destinoAposEntrar('grupo', 'https://outro.site')).toBe(ROTA_PADRAO)
    expect(destinoAposEntrar('grupo', 'http://outro.site')).toBe(ROTA_PADRAO)
  })

  // O navegador trata `//host` como externo.
  it('recusa caminho protocolo-relativo', () => {
    expect(destinoAposEntrar('grupo', '//outro.site/x')).toBe(ROTA_PADRAO)
  })

  it('redirect vazio ou nulo cai na rota do contexto', () => {
    expect(destinoAposEntrar('plataforma', '')).toBe(ROTA_PLATAFORMA)
    expect(destinoAposEntrar('plataforma', null)).toBe(ROTA_PLATAFORMA)
  })
})
