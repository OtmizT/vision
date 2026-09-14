import type { ChartConfiguration, ChartType } from 'chart.js'

/**
 * Os quatro tipos que o assistente pode pedir, e o formato dos dados.
 *
 * O alias existe por uma razão de compilação, não de estilo: `ChartConfiguration`
 * sem argumentos assume `keyof ChartTypeRegistry`, o que obriga o TypeScript a
 * expandir a união sobre TODOS os tipos de gráfico registrados. Essa expansão
 * transborda para outros arquivos — com ela, `let chart: Chart | null` em
 * GraficoDonut e DashboardView passa a recusar o `new Chart()` que sempre
 * aceitou, com um erro de variância que nada tem a ver com este módulo.
 *
 * Nomear a instanciação concreta mantém o custo de inferência local.
 */
type ConfigVisao = ChartConfiguration<'bar' | 'line' | 'doughnut', number[], string>
import { corDe, corDeAlpha } from './paleta'
import { coresGrafico, comAlfa } from './tema'

/**
 * Especificação do assistente → configuração do Chart.js.
 *
 * A IA **não** devolve config do Chart.js. Ela devolve o que desenhar — tipo,
 * rótulos, séries — e a conversão acontece aqui, com a paleta, as fontes e as
 * cores de tema da aplicação.
 *
 * Duas razões, e as duas importam:
 *
 *   visual  um gráfico montado pelo modelo sairia com as cores que ele
 *           inventasse, e o chat pareceria outro produto dentro do produto.
 *   risco   config do Chart.js é um objeto que o navegador executa; passar
 *           JSON gerado por modelo direto para `new Chart()` é entregar a
 *           configuração de um componente a uma fonte não confiável.
 *
 * O servidor já valida a spec (internal/ia/spec.go). A validação daqui é a
 * segunda: o frontend não confia em ter recebido o que espera.
 */

export interface SerieSpec {
  nome: string
  valores: number[]
}

export interface VisaoSpec {
  titulo: string
  tipo: 'barra' | 'barra_horizontal' | 'linha' | 'rosca'
  rotulos: string[]
  series: SerieSpec[]
  formato: 'moeda' | 'numero'
}

const TIPOS: Record<VisaoSpec['tipo'], ChartType> = {
  barra: 'bar',
  barra_horizontal: 'bar',
  linha: 'line',
  rosca: 'doughnut',
}

/**
 * Valida a spec antes de virar gráfico.
 *
 * A checagem que mais importa é o alinhamento entre série e rótulos: o Chart.js
 * desenha uma série mais curta sem reclamar, casando pelo índice — e a barra de
 * março passa a mostrar o valor de abril. Ninguém percebe olhando, e é um
 * gráfico sobre dinheiro.
 */
export function specValida(s: unknown): s is VisaoSpec {
  if (!s || typeof s !== 'object') return false
  const v = s as Partial<VisaoSpec>

  if (typeof v.tipo !== 'string' || !(v.tipo in TIPOS)) return false
  if (!Array.isArray(v.rotulos) || v.rotulos.length === 0) return false
  if (!Array.isArray(v.series) || v.series.length === 0) return false

  return v.series.every(
    serie =>
      serie &&
      Array.isArray(serie.valores) &&
      serie.valores.length === v.rotulos!.length &&
      serie.valores.every(n => typeof n === 'number' && Number.isFinite(n)),
  )
}

const fmtMoeda = new Intl.NumberFormat('pt-BR', {
  style: 'currency', currency: 'BRL', maximumFractionDigits: 0,
})
const fmtNumero = new Intl.NumberFormat('pt-BR', { maximumFractionDigits: 0 })

function formatar(v: number, formato: VisaoSpec['formato']): string {
  return formato === 'numero' ? fmtNumero.format(v) : fmtMoeda.format(v)
}

/**
 * Monta a configuração. Devolve `null` para spec inválida — desenhar um gráfico
 * errado é pior que não desenhar: o errado parece certo.
 *
 * `raiz` existe para o teste poder injetar tokens; em produção lê do documento.
 */
export function configDoGrafico(entrada: unknown, raiz?: Element): ConfigVisao | null {
  if (!specValida(entrada)) return null

  // Const e não o parâmetro: o estreitamento de tipo do guard não atravessa
  // funções aninhadas (escalas(), abaixo), porque um parâmetro poderia ser
  // reatribuído no meio do caminho.
  const spec = entrada

  const c = coresGrafico(raiz)
  const horizontal = spec.tipo === 'barra_horizontal'
  const rosca = spec.tipo === 'rosca'

  /*
   * Cores. Na rosca cada FATIA é uma categoria, então a cor varia por ponto;
   * nos demais cada SÉRIE é uma categoria, e a cor varia por série. Trocar isso
   * produz um gráfico de barras multicolorido em que a cor não significa nada.
   */
  const datasets = spec.series.map((serie, i) => ({
    label: serie.nome,
    data: serie.valores,
    backgroundColor: rosca
      ? spec.rotulos.map((_, j) => corDe(j))
      : spec.tipo === 'linha'
        ? corDeAlpha(i, 0.15)
        : corDe(i),
    borderColor: rosca ? c.fundo : corDe(i),
    borderWidth: rosca ? 2 : spec.tipo === 'linha' ? 2 : 0,
    borderRadius: rosca || spec.tipo === 'linha' ? 0 : 4,
    fill: spec.tipo === 'linha',
    tension: 0.3,
    pointRadius: 3,
  }))

  /*
   * O eixo de VALOR recebe o formatador; o eixo de CATEGORIA não recebe nada.
   *
   * A chave `callback` é omitida em vez de receber `undefined`. Passá-la como
   * undefined não é o mesmo que não passá-la: o Chart.js trata a propriedade
   * como definida e deixa de usar o callback padrão, que é justamente quem
   * converte o índice no rótulo. O eixo passa a mostrar 0, 1, 2 em vez dos
   * nomes — foi o que apareceu na verificação no navegador.
   */
  function escalas() {
    const eixoValor = {
      grid: { color: c.grade },
      ticks: {
        color: c.tick,
        font: { family: c.fonte, size: 10 },
        callback: (v: string | number) => formatar(Number(v), spec.formato),
      },
    }
    const eixoCategoria = {
      grid: { color: 'transparent' },
      ticks: { color: c.tick, font: { family: c.fonte, size: 10 } },
    }
    return horizontal
      ? { x: eixoValor, y: eixoCategoria }
      : { x: eixoCategoria, y: eixoValor }
  }

  return {
    type: TIPOS[spec.tipo],
    data: { labels: spec.rotulos, datasets },
    options: {
      responsive: true,
      // O contêiner manda na altura. Sem isto o canvas realimenta o pai e a
      // bolha do chat cresce sem parar.
      maintainAspectRatio: false,
      indexAxis: horizontal ? 'y' : 'x',
      plugins: {
        // Uma série só já é nomeada pelo título; a legenda seria repetição
        // ocupando espaço numa bolha estreita.
        legend: {
          display: spec.series.length > 1 || rosca,
          position: rosca ? 'bottom' : 'top',
          labels: { color: c.rotulo, font: { family: c.fonte, size: 11 }, boxWidth: 10 },
        },
        // Números em cima de cada barra poluem uma bolha de 380px. O valor
        // aparece no tooltip.
        datalabels: { display: false },
        tooltip: {
          backgroundColor: c.tooltipFundo,
          borderColor: c.tooltipBorda,
          borderWidth: 1,
          titleColor: c.tooltipTexto,
          bodyColor: c.tooltipTexto,
          titleFont: { family: c.fonte },
          bodyFont: { family: c.fonte },
          callbacks: {
            label: (item: { dataset: { label?: string }; parsed: { x: number; y: number } }) => {
              const v = horizontal ? item.parsed.x : item.parsed.y
              const nome = item.dataset.label ? `${item.dataset.label}: ` : ''
              return nome + formatar(v, spec.formato)
            },
          },
        },
      },
      scales: rosca ? undefined : escalas(),
    },
  } as ConfigVisao
}

export { comAlfa }
