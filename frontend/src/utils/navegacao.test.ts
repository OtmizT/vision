import { describe, it, expect } from 'vitest'
import {
  rotaInicial, temDashboard, destinoAposEntrar,
  ROTA_ADMIN_GLOBAL, ROTA_PADRAO, type Papel,
} from './navegacao'

const PAPEIS: Papel[] = ['admin_global', 'admin_grupo', 'viewer']

describe('rotaInicial', () => {
  it('admin global vai para o Sync Control', () => {
    expect(rotaInicial('admin_global')).toBe(ROTA_ADMIN_GLOBAL)
  })

  it('admin de grupo e viewer vão para o Dashboard', () => {
    expect(rotaInicial('admin_grupo')).toBe(ROTA_PADRAO)
    expect(rotaInicial('viewer')).toBe(ROTA_PADRAO)
  })

  // O redirect roda no guard antes de `auth.user` existir em alguns caminhos.
  it('papel ausente cai no padrão, não em undefined', () => {
    expect(rotaInicial(undefined)).toBe(ROTA_PADRAO)
    expect(rotaInicial(null)).toBe(ROTA_PADRAO)
  })

  // Mandar alguém para uma tela que ele não pode ver rende 403 sem explicação.
  it('todo papel tem destino, e o destino é um caminho interno', () => {
    for (const p of PAPEIS) {
      expect(rotaInicial(p).startsWith('/')).toBe(true)
    }
  })
})

describe('temDashboard', () => {
  it('admin global não tem', () => {
    expect(temDashboard('admin_global')).toBe(false)
  })

  it('os demais têm', () => {
    expect(temDashboard('admin_grupo')).toBe(true)
    expect(temDashboard('viewer')).toBe(true)
    expect(temDashboard(undefined)).toBe(true)
  })

  // Menu e roteador precisam concordar: discordando, ou o item some e a URL
  // continua aberta, ou o item aparece e leva a um 403.
  it('quem não tem Dashboard não é mandado para ele', () => {
    for (const p of PAPEIS) {
      if (!temDashboard(p)) expect(rotaInicial(p)).not.toBe(ROTA_PADRAO)
    }
  })
})

describe('destinoAposEntrar', () => {
  it('sem redirect, usa a rota do papel', () => {
    expect(destinoAposEntrar('admin_global')).toBe(ROTA_ADMIN_GLOBAL)
    expect(destinoAposEntrar('viewer')).toBe(ROTA_PADRAO)
  })

  it('honra um redirect interno', () => {
    expect(destinoAposEntrar('admin_grupo', '/usuarios')).toBe('/usuarios')
  })

  // Um link com redirect externo levaria o usuário recém-autenticado para fora.
  it('recusa redirect para outro site', () => {
    expect(destinoAposEntrar('viewer', 'https://outro.site')).toBe(ROTA_PADRAO)
    expect(destinoAposEntrar('viewer', 'http://outro.site')).toBe(ROTA_PADRAO)
  })

  // O navegador trata `//host` como externo.
  it('recusa caminho protocolo-relativo', () => {
    expect(destinoAposEntrar('viewer', '//outro.site/x')).toBe(ROTA_PADRAO)
  })

  it('redirect vazio ou nulo cai na rota do papel', () => {
    expect(destinoAposEntrar('admin_global', '')).toBe(ROTA_ADMIN_GLOBAL)
    expect(destinoAposEntrar('admin_global', null)).toBe(ROTA_ADMIN_GLOBAL)
  })
})
