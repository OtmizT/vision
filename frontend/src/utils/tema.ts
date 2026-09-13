/**
 * Ponte entre os tokens CSS e o Chart.js.
 *
 * O canvas não entende uma referência de token: a string chega crua ao motor de
 * desenho e é descartada como cor inválida. Antes deste módulo o dashboard passava
 * a referência crua do token como preenchimento e como família de fonte —
 * nenhuma das duas surtia efeito. Aqui os tokens são resolvidos para
 * valores concretos antes de chegarem ao gráfico.
 *
 * A leitura acontece a cada montagem de gráfico, e não uma vez no import: o
 * tema muda em tempo de execução e um cache guardaria as cores do tema antigo.
 */

/** Lê um token do elemento raiz. `alternativa` cobre SSR e testes sem CSS. */
export function token(nome: string, alternativa: string, raiz?: Element): string {
  const el = raiz ?? (typeof document !== 'undefined' ? document.documentElement : null)
  if (!el) return alternativa
  const v = getComputedStyle(el).getPropertyValue(nome).trim()
  return v || alternativa
}

/**
 * Converte `#rgb`/`#rrggbb` em `rgba(...)`. Necessário porque o Chart.js precisa
 * de preenchimentos translúcidos e os tokens guardam a cor opaca.
 * Valor não reconhecido volta inalterado — melhor uma cor sólida que nenhuma.
 */
export function comAlfa(cor: string, alfa: number): string {
  const m = cor.trim().match(/^#([0-9a-f]{3}|[0-9a-f]{6})$/i)
  if (!m) return cor
  let h = m[1]
  if (h.length === 3) h = h[0] + h[0] + h[1] + h[1] + h[2] + h[2]
  const n = parseInt(h, 16)
  return `rgba(${(n >> 16) & 255}, ${(n >> 8) & 255}, ${n & 255}, ${alfa})`
}

export interface CoresGrafico {
  receita: string
  despesa: string
  linha: string
  /** Cor de posição — saldo e total da cascata. Distinta de receita e despesa,
      que marcam variação. */
  acento: string
  grade: string
  tick: string
  rotulo: string
  fundo: string
  tooltipFundo: string
  tooltipBorda: string
  tooltipTexto: string
  fonte: string
}

/**
 * Paleta resolvida para os gráficos. Cada alternativa repete o valor do token
 * no tema escuro: se o CSS não tiver carregado, o gráfico sai com a cor certa
 * em vez de preto.
 */
export function coresGrafico(raiz?: Element): CoresGrafico {
  return {
    receita:      token('--success',      '#22c55e', raiz),
    despesa:      token('--danger',       '#ef4444', raiz),
    linha:        token('--primary',      '#7c4ddc', raiz),
    acento:       token('--accent',       '#22d3ee', raiz),
    grade:        token('--chart-grid',   'rgba(255,255,255,0.04)', raiz),
    tick:         token('--text-dim',     '#7a90a8', raiz),
    rotulo:       token('--text-muted',   '#a8b8cc', raiz),
    fundo:        token('--surface',      '#0d1520', raiz),
    tooltipFundo: token('--surface-2',    '#131c2b', raiz),
    tooltipBorda: token('--border-strong','rgba(255,255,255,0.12)', raiz),
    tooltipTexto: token('--text',         '#e2eaf4', raiz),
    // Texto DENTRO do canvas — rótulos de eixo e datalabels. Mesma família do
    // resto da aplicação; o Chart.js precisa da string resolvida porque o canvas
    // não entende a referência do token.
    fonte: token('--font-body', 'Geist Variable, system-ui, sans-serif', raiz),
  }
}
