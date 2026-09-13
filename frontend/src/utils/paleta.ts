/**
 * Paleta categórica dos gráficos de composição.
 *
 * Doze cores porque acima disso a distinção visual deixa de ser confiável;
 * passando disso a paleta se repete, e a legenda lateral é que desfaz a
 * ambiguidade.
 *
 * ── Como estas doze foram escolhidas ────────────────────────────────────────
 *
 * Não no olho. A paleta anterior abria num ciano néon e seguia por tons
 * saturados de biblioteca; o registro destoava de um produto financeiro. Estas
 * foram construídas em OKLCH sob quatro restrições medidas, e a ordem foi
 * otimizada por busca local:
 *
 *   matiz     os cinco tons escolhidos entram com o matiz EXATO pedido —
 *             azul-aço 221°, lavanda 295°, malva 350°, dourado 78°, teal 171°.
 *             Os outros sete preenchem os vãos entre eles.
 *   croma     C ≈ 0.125, o registro sóbrio dos tons de referência. É um teto,
 *             não um alvo a maximizar: croma livre devolve néon de volta.
 *   tom       L ≈ 0.66, a interseção das faixas que os dois temas admitem.
 *             Os tons de referência vinham em L ≈ 0.73 e precisaram escurecer:
 *             naquela altura a cor não se sustenta sobre o card branco, rendia
 *             2.0-2.3:1, abaixo do mínimo de 3:1 para elemento gráfico.
 *   distância cada par VIZINHO no array se separa o suficiente na visão normal
 *             (ΔE ≥ 15) e nas três formas de daltonismo (ΔE ≥ 8). Vizinho no
 *             array é vizinho na tela: são as fatias que se tocam no donut.
 *
 * Medido contra as superfícies reais de cada tema — o `--surface` do card, não
 * o fundo da página. Todas as doze passam em ambos, então não há paleta por
 * tema: os componentes de gráfico não observam a troca de tema, e uma paleta
 * dupla exigiria essa reatividade sem ganho proporcional.
 *
 * A primeira posição é fixa: ela pinta a MAIOR fatia. Verde ali faria "maior
 * categoria de despesa" parecer uma boa notícia.
 *
 * O teste em paleta.test.ts refaz estas contas. Trocar uma cor no olho quebra o
 * teste, que é o ponto.
 */
export const PALETA = [
  '#00a2c6', // azul-aço   221°
  '#9a9930', // oliva      109°
  '#cd739f', // malva      350°
  '#6395e1', // azul       258°
  '#d6727c', // rosa-tijolo 15°
  '#06a5a7', // turquesa   196°
  '#be8a29', // dourado     78°
  '#9a84d9', // lavanda    295°
  '#13a886', // teal       171°
  '#d27b4d', // terracota   47°
  '#b879c2', // orquídea   322°
  '#65a358', // verde      140°
] as const

export function corDe(indice: number): string {
  return PALETA[indice % PALETA.length]
}

/** Mesma cor com alfa, para preenchimentos. */
export function corDeAlpha(indice: number, alpha: number): string {
  const hex = corDe(indice)
  const n = parseInt(hex.slice(1), 16)
  return `rgba(${(n >> 16) & 255}, ${(n >> 8) & 255}, ${n & 255}, ${alpha})`
}
