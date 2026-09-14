import { describe, it, expect } from 'vitest'
import { configDoGrafico, specValida, type VisaoSpec } from './visaospec'
import { PALETA } from './paleta'

function spec(over: Partial<VisaoSpec> = {}): VisaoSpec {
  return {
    titulo: 'Receita por mês',
    tipo: 'barra',
    rotulos: ['Jan', 'Fev', 'Mar'],
    series: [{ nome: 'Receita', valores: [100, 200, 300] }],
    formato: 'moeda',
    ...over,
  }
}

describe('specValida', () => {
  it('aceita uma spec bem formada', () => {
    expect(specValida(spec())).toBe(true)
  })

  /*
   * O defeito que esta validação existe para impedir.
   *
   * O Chart.js desenha uma série mais curta sem reclamar, casando pelo índice —
   * e a barra de março passa a mostrar o valor de abril. Ninguém percebe
   * olhando, e é um gráfico sobre dinheiro.
   */
  it('recusa série desalinhada dos rótulos', () => {
    expect(specValida(spec({ series: [{ nome: 'x', valores: [1, 2] }] }))).toBe(false)
    expect(specValida(spec({ series: [{ nome: 'x', valores: [1, 2, 3, 4] }] }))).toBe(false)
  })

  // NaN e Infinity viram buraco no gráfico ou escala infinita.
  it('recusa valores que não são número finito', () => {
    expect(specValida(spec({ series: [{ nome: 'x', valores: [1, NaN, 3] }] }))).toBe(false)
    expect(specValida(spec({ series: [{ nome: 'x', valores: [1, Infinity, 3] }] }))).toBe(false)
    expect(specValida(spec({ series: [{ nome: 'x', valores: [1, '2' as never, 3] }] }))).toBe(false)
  })

  it('recusa o que não é spec', () => {
    for (const lixo of [null, undefined, 'texto', 42, [], {}]) {
      expect(specValida(lixo)).toBe(false)
    }
  })

  it('recusa tipo que o frontend não sabe desenhar', () => {
    expect(specValida(spec({ tipo: 'pizza3d' as never }))).toBe(false)
  })

  it('recusa gráfico sem eixo ou sem dado', () => {
    expect(specValida(spec({ rotulos: [] }))).toBe(false)
    expect(specValida(spec({ series: [] }))).toBe(false)
  })
})

describe('configDoGrafico', () => {
  // Desenhar errado é pior que não desenhar: o errado parece certo.
  it('spec inválida não vira gráfico', () => {
    expect(configDoGrafico(null)).toBeNull()
    expect(configDoGrafico(spec({ series: [{ nome: 'x', valores: [1] }] }))).toBeNull()
  })

  it('usa a paleta da aplicação, não cor inventada pelo modelo', () => {
    const cfg = configDoGrafico(spec())!
    expect(cfg.data.datasets[0].backgroundColor).toBe(PALETA[0])
  })

  /*
   * Na rosca cada FATIA é uma categoria, então a cor varia por ponto. Nos
   * demais cada SÉRIE é uma categoria. Trocar isso dá um gráfico de barras
   * multicolorido em que a cor não significa nada.
   */
  it('rosca colore por fatia; barra colore por série', () => {
    const rosca = configDoGrafico(spec({ tipo: 'rosca' }))!
    expect(Array.isArray(rosca.data.datasets[0].backgroundColor)).toBe(true)
    expect(rosca.data.datasets[0].backgroundColor).toEqual([PALETA[0], PALETA[1], PALETA[2]])

    const barra = configDoGrafico(spec())!
    expect(Array.isArray(barra.data.datasets[0].backgroundColor)).toBe(false)
  })

  it('duas séries recebem cores diferentes', () => {
    const cfg = configDoGrafico(spec({
      series: [
        { nome: 'Receita', valores: [1, 2, 3] },
        { nome: 'Despesa', valores: [4, 5, 6] },
      ],
    }))!
    expect(cfg.data.datasets[0].backgroundColor).not.toBe(cfg.data.datasets[1].backgroundColor)
  })

  it('barra horizontal inverte o eixo de índice', () => {
    expect(configDoGrafico(spec({ tipo: 'barra_horizontal' }))!.options!.indexAxis).toBe('y')
    expect(configDoGrafico(spec())!.options!.indexAxis).toBe('x')
  })

  it('mapeia os quatro tipos para o Chart.js', () => {
    expect(configDoGrafico(spec({ tipo: 'barra' }))!.type).toBe('bar')
    expect(configDoGrafico(spec({ tipo: 'barra_horizontal' }))!.type).toBe('bar')
    expect(configDoGrafico(spec({ tipo: 'linha' }))!.type).toBe('line')
    expect(configDoGrafico(spec({ tipo: 'rosca' }))!.type).toBe('doughnut')
  })

  // Sem isto o canvas realimenta o contêiner e a bolha do chat cresce sem parar.
  it('não mantém proporção — quem manda na altura é a bolha', () => {
    expect(configDoGrafico(spec())!.options!.maintainAspectRatio).toBe(false)
  })

  // Uma série só já é nomeada pelo título da bolha; a legenda seria repetição
  // ocupando espaço num painel estreito.
  it('legenda some com uma série e aparece com duas', () => {
    const uma = configDoGrafico(spec())!
    expect(uma.options!.plugins!.legend!.display).toBe(false)

    const duas = configDoGrafico(spec({
      series: [
        { nome: 'a', valores: [1, 2, 3] },
        { nome: 'b', valores: [1, 2, 3] },
      ],
    }))!
    expect(duas.options!.plugins!.legend!.display).toBe(true)
  })

  // Na rosca a legenda é o que nomeia as fatias — precisa aparecer mesmo com
  // uma série só.
  it('rosca sempre mostra legenda', () => {
    expect(configDoGrafico(spec({ tipo: 'rosca' }))!.options!.plugins!.legend!.display).toBe(true)
  })

  it('rosca não tem escalas', () => {
    expect(configDoGrafico(spec({ tipo: 'rosca' }))!.options!.scales).toBeUndefined()
    expect(configDoGrafico(spec())!.options!.scales).toBeDefined()
  })

  it('formata valores em reais no eixo', () => {
    const cfg = configDoGrafico(spec())!
    const cb = cfg.options!.scales!.y!.ticks!.callback as (v: number) => string
    const saida = cb(1500)
    expect(saida).toContain('R$')
    expect(saida).toContain('1.500')
  })

  /*
   * O defeito que a verificação no navegador pegou.
   *
   * `callback: undefined` NÃO é o mesmo que não passar a chave: o Chart.js trata
   * a propriedade como definida e para de usar o callback padrão, que é quem
   * converte o índice no rótulo. O eixo passava a mostrar 0, 1, 2 no lugar dos
   * nomes dos clientes.
   */
  it('o eixo de categoria não recebe callback — senão mostra índice no lugar do nome', () => {
    const vertical = configDoGrafico(spec())!
    expect('callback' in (vertical.options!.scales!.x!.ticks ?? {})).toBe(false)

    const horizontal = configDoGrafico(spec({ tipo: 'barra_horizontal' }))!
    expect('callback' in (horizontal.options!.scales!.y!.ticks ?? {})).toBe(false)
  })

  it('o eixo de valor troca de lado na barra horizontal', () => {
    // Vertical: os valores estão em y. Horizontal: em x. Formatar o eixo errado
    // poria "R$ 1.240.000" onde deveria estar o nome do cliente.
    const vertical = configDoGrafico(spec())!
    expect(typeof vertical.options!.scales!.y!.ticks!.callback).toBe('function')

    const horizontal = configDoGrafico(spec({ tipo: 'barra_horizontal' }))!
    expect(typeof horizontal.options!.scales!.x!.ticks!.callback).toBe('function')
  })

  it('formato numero não coloca R$', () => {
    const cfg = configDoGrafico(spec({ formato: 'numero' }))!
    const cb = cfg.options!.scales!.y!.ticks!.callback as (v: number) => string
    expect(cb(1500)).not.toContain('R$')
  })
})
