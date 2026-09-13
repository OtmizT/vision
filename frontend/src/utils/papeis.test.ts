import { describe, it, expect } from 'vitest'
import { papeisAtribuiveis, rotuloPapel } from './papeis'

describe('papeisAtribuiveis', () => {
  /*
   * O seletor oferecia "Admin Global" para quem já era admin global, e o backend
   * gravava — por uma rota de grupo — o papel que manda na plataforma inteira.
   * Hoje o backend devolve 422 e o banco recusa: a opção não pode voltar aqui.
   */
  it('não oferece admin_global', () => {
    expect(papeisAtribuiveis().map(p => p.value)).not.toContain('admin_global')
  })

  it('oferece os dois papéis de grupo', () => {
    expect(papeisAtribuiveis().map(p => p.value)).toEqual(['admin_grupo', 'viewer'])
  })

  it('toda opção tem rótulo', () => {
    for (const o of papeisAtribuiveis()) expect(o.label).toBeTruthy()
  })

  // Mutar a lista devolvida não pode contaminar a próxima chamada.
  it('cada chamada devolve uma lista própria', () => {
    const a = papeisAtribuiveis()
    a.push({ value: 'viewer', label: 'X' })
    expect(papeisAtribuiveis()).toHaveLength(2)
  })
})

describe('rotuloPapel', () => {
  // A conta existe e aparece na lista do grupo dela; esconder o papel faria a
  // lista mentir. Não atribuir é diferente de não exibir.
  it('admin_global continua tendo rótulo', () => {
    expect(rotuloPapel('admin_global')).toBe('Admin Global')
  })

  it('papéis de grupo têm rótulo legível', () => {
    expect(rotuloPapel('admin_grupo')).toBe('Admin Grupo')
    expect(rotuloPapel('viewer')).toBe('Viewer')
  })

  // Papel desconhecido vindo de um banco antigo não pode virar vazio na tela.
  it('papel desconhecido aparece como veio', () => {
    expect(rotuloPapel('operador')).toBe('operador')
    expect(rotuloPapel('')).toBe('')
  })
})
