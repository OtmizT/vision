import { describe, it, expect } from 'vitest'
import { PALETA, corDe, corDeAlpha } from './paleta'

/*
 * Contrato da paleta dos gráficos.
 *
 * As doze cores foram construídas sob restrições medidas (ver paleta.ts), e sem
 * teste essas restrições existiriam só num comentário: bastava alguém colar um
 * hex "mais bonito" para a paleta voltar ao registro néon de onde saiu, ou para
 * duas fatias vizinhas do donut ficarem indistinguíveis.
 *
 * A matemática é refeita aqui em vez de importada porque o cálculo é o teste:
 * uma dependência que sumisse levaria junto a verificação.
 */

// Superfícies reais onde os gráficos são desenhados — o card, não o fundo da
// página. --surface em assets/tokens.css.
const SUPERFICIES = { claro: '#ffffff', escuro: '#0f1230' }

// ── conversões de cor ────────────────────────────────────────────────────────

const s2lin = (c: number) => (c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4)

function rgbLinear(hex: string): [number, number, number] {
  const h = hex.replace('#', '')
  return [0, 2, 4].map(i => s2lin(parseInt(h.slice(i, i + 2), 16) / 255)) as [number, number, number]
}

/** Luminância relativa da WCAG. */
function luminancia(hex: string): number {
  const [r, g, b] = rgbLinear(hex)
  return 0.2126 * r + 0.7152 * g + 0.0722 * b
}

function contraste(a: string, b: string): number {
  const [alto, baixo] = [luminancia(a), luminancia(b)].sort((x, y) => y - x)
  return (alto + 0.05) / (baixo + 0.05)
}

/** OKLab: espaço perceptualmente uniforme, onde distância euclidiana ≈ diferença percebida. */
function oklab([r, g, b]: [number, number, number]): [number, number, number] {
  const l = Math.cbrt(0.4122214708 * r + 0.5363325363 * g + 0.0514459929 * b)
  const m = Math.cbrt(0.2119034982 * r + 0.6806995451 * g + 0.1073969566 * b)
  const s = Math.cbrt(0.0883024619 * r + 0.2817188376 * g + 0.6299787005 * b)
  return [
    0.2104542553 * l + 0.793617785 * m - 0.0040720468 * s,
    1.9779984951 * l - 2.428592205 * m + 0.4505937099 * s,
    0.0259040371 * l + 0.7827717662 * m - 0.808675766 * s,
  ]
}

/** Matrizes de Machado et al. para simular as três dicromacias. */
const DALTONISMO = {
  deutan: [[0.367322, 0.860646, -0.227968], [0.280085, 0.672501, 0.047413], [-0.01182, 0.04294, 0.968881]],
  protan: [[0.152286, 1.052583, -0.204868], [0.114503, 0.786281, 0.099216], [-0.003882, -0.048116, 1.051998]],
  tritan: [[1.255528, -0.076749, -0.178779], [-0.078411, 0.930809, 0.147602], [0.004733, 0.691367, 0.3039]],
} as const

type Visao = keyof typeof DALTONISMO | 'normal'

function comoSeVe(hex: string, visao: Visao): [number, number, number] {
  const rgb = rgbLinear(hex)
  if (visao === 'normal') return rgb
  const M = DALTONISMO[visao]
  return M.map(linha => Math.max(0, Math.min(1, linha[0] * rgb[0] + linha[1] * rgb[1] + linha[2] * rgb[2]))) as [number, number, number]
}

/** Distância percebida entre duas cores, em OKLab ×100. */
function distancia(a: string, b: string, visao: Visao): number {
  const [l1, a1, b1] = oklab(comoSeVe(a, visao))
  const [l2, a2, b2] = oklab(comoSeVe(b, visao))
  return 100 * Math.hypot(l1 - l2, a1 - a2, b1 - b2)
}

function croma(hex: string): number {
  const [, a, b] = oklab(rgbLinear(hex))
  return Math.hypot(a, b)
}

function tom(hex: string): number {
  return oklab(rgbLinear(hex))[0]
}

// ── testes ───────────────────────────────────────────────────────────────────

describe('paleta: forma', () => {
  it('tem doze cores, todas em hex de seis dígitos', () => {
    expect(PALETA).toHaveLength(12)
    for (const cor of PALETA) expect(cor).toMatch(/^#[0-9a-f]{6}$/)
  })

  it('nenhuma cor se repete', () => {
    expect(new Set(PALETA).size).toBe(PALETA.length)
  })

  // Ela pinta a maior fatia do donut. Verde ali faria "maior categoria de
  // despesa" parecer uma boa notícia.
  it('a primeira cor é o azul-aço, e não um tom com carga semântica', () => {
    expect(PALETA[0]).toBe('#00a2c6')
  })
})

describe('paleta: registro sóbrio', () => {
  /*
   * O motivo de a paleta ter sido refeita. A anterior abria em #00e5ff, um
   * ciano néon de croma ~0.17 — vibrante demais para relatório financeiro.
   */
  it('nenhuma cor é saturada a ponto de vibrar', () => {
    const vibrantes = PALETA.filter(c => croma(c) > 0.14).map(c => `${c} (C=${croma(c).toFixed(3)})`)
    expect(vibrantes).toEqual([])
  })

  // Croma baixo demais vira cinza, e doze cinzas não são doze categorias.
  it('nenhuma cor é lavada a ponto de virar cinza', () => {
    const lavadas = PALETA.filter(c => croma(c) < 0.1).map(c => `${c} (C=${croma(c).toFixed(3)})`)
    expect(lavadas).toEqual([])
  })

  /*
   * Faixa de tom que os dois temas admitem ao mesmo tempo. Acima de 0.67 a cor
   * some no card branco; abaixo de 0.48, no fundo escuro.
   */
  it('todas ficam na faixa de tom que serve aos dois temas', () => {
    const fora = PALETA.filter(c => tom(c) < 0.48 || tom(c) > 0.68).map(c => `${c} (L=${tom(c).toFixed(3)})`)
    expect(fora).toEqual([])
  })
})

describe('paleta: legibilidade nos dois temas', () => {
  /*
   * 3:1 é o mínimo da WCAG 1.4.11 para elemento gráfico não-textual — uma fatia
   * de donut é exatamente isso. Os tons originais sugeridos (#4BB8D8, #4EC9A6)
   * rendiam 2.0-2.3:1 sobre branco e foram escurecidos por causa desta linha.
   */
  for (const [nome, fundo] of Object.entries(SUPERFICIES)) {
    it(`toda cor se destaca do fundo no tema ${nome}`, () => {
      const fracas = PALETA
        .map(c => [c, contraste(c, fundo)] as const)
        .filter(([, r]) => r < 3)
        .map(([c, r]) => `${c} (${r.toFixed(2)}:1)`)
      expect(fracas).toEqual([])
    })
  }
})

describe('paleta: fatias vizinhas se distinguem', () => {
  /*
   * Vizinho no array é vizinho na tela: são as fatias que se tocam no donut e as
   * barras que ficam lado a lado. Os pares NÃO adjacentes podem se parecer, e é
   * por isso que a ORDEM do array faz parte do desenho — trocar duas posições
   * pode quebrar este teste sem trocar cor nenhuma.
   */
  const vizinhos = PALETA.slice(0, -1).map((c, i) => [c, PALETA[i + 1]] as const)

  it('na visão normal, nenhum par vizinho é quase igual', () => {
    const proximos = vizinhos
      .map(([a, b]) => [a, b, distancia(a, b, 'normal')] as const)
      .filter(([, , d]) => d < 15)
      .map(([a, b, d]) => `${a}↔${b} (ΔE ${d.toFixed(1)})`)
    expect(proximos).toEqual([])
  })

  // ~8% dos homens têm alguma dicromacia. Num painel de contas a pagar, duas
  // fatias que colapsam é uma leitura errada de para onde o dinheiro foi.
  for (const visao of ['deutan', 'protan', 'tritan'] as const) {
    it(`sob ${visao}, nenhum par vizinho colapsa`, () => {
      const proximos = vizinhos
        .map(([a, b]) => [a, b, distancia(a, b, visao)] as const)
        .filter(([, , d]) => d < 8)
        .map(([a, b, d]) => `${a}↔${b} (ΔE ${d.toFixed(1)})`)
      expect(proximos).toEqual([])
    })
  }
})

describe('corDe', () => {
  it('devolve a cor de cada índice, na ordem', () => {
    PALETA.forEach((cor, i) => expect(corDe(i)).toBe(cor))
  })

  // Mais de doze categorias é o caso comum num plano de contas real.
  it('dá a volta depois da última', () => {
    expect(corDe(12)).toBe(PALETA[0])
    expect(corDe(25)).toBe(PALETA[1])
  })

  it('nunca devolve indefinido', () => {
    for (let i = 0; i < 120; i++) expect(corDe(i)).toMatch(/^#[0-9a-f]{6}$/)
  })
})

describe('corDeAlpha', () => {
  it('produz rgba com o canal pedido', () => {
    expect(corDeAlpha(0, 0.18)).toBe('rgba(0, 162, 198, 0.18)')
  })

  it('preserva o RGB da cor de origem', () => {
    for (let i = 0; i < PALETA.length; i++) {
      const [, r, g, b] = corDeAlpha(i, 1).match(/rgba\((\d+), (\d+), (\d+),/)!
      const hex = '#' + [r, g, b].map(v => Number(v).toString(16).padStart(2, '0')).join('')
      expect(hex).toBe(corDe(i))
    }
  })

  it('dá a volta como corDe', () => {
    expect(corDeAlpha(12, 0.5)).toBe(corDeAlpha(0, 0.5))
  })
})
