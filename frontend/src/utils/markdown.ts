import { marked } from 'marked'
import DOMPurify from 'dompurify'

/**
 * Markdown do assistente → HTML seguro.
 *
 * ── Por que isto existe, e por que não é detalhe de implementação ──
 *
 * Este é o PRIMEIRO HTML dinâmico do projeto: até aqui não havia um único
 * `v-html` em lugar nenhum, e a superfície de XSS era zero. O que se ganha em
 * legibilidade — negrito nos números, listas, tabelas — se paga com essa
 * superfície, então ela precisa ser fechada aqui e testada.
 *
 * E o caminho de ataque não é teórico. `cliente_final` e
 * `descricao_categoria` vêm do ERP da Omie, preenchidos PELO CLIENTE. Uma
 * categoria nomeada `<img src=x onerror="...">` viajaria assim:
 *
 *   cliente nomeia a categoria  →  sync traz para o schema do tenant
 *   →  a ferramenta devolve o nome à IA  →  a IA ecoa na resposta
 *   →  o navegador do operador da OTM executa
 *
 * E o `access_token` mora no localStorage. A vítima seria alguém com acesso
 * administrativo aos dados de dezenas de empresas.
 *
 * A pseudonimização NÃO cobre isso: ela troca nomes de cliente, não descrições
 * de categoria.
 *
 * ── A defesa ──
 *
 * O modelo devolve Markdown, não HTML, então `<script>` nem chega a ser uma tag
 * — vira texto. E o resultado ainda passa por uma lista de permissão estrita,
 * porque Markdown aceita HTML embutido e porque o dia em que alguém trocar o
 * parser esta linha continua valendo.
 */

/** As únicas tags que podem sair daqui. Sem img, iframe, form, style, svg. */
const TAGS_PERMITIDAS = [
  'p', 'br', 'hr',
  'strong', 'b', 'em', 'i', 'del', 's',
  'ul', 'ol', 'li',
  'blockquote',
  'code', 'pre',
  'table', 'thead', 'tbody', 'tr', 'th', 'td',
  'h1', 'h2', 'h3', 'h4',
  'a',
  'span',
]

/**
 * Nenhum atributo `on*`, nenhum `style`.
 *
 * `href` é permitido, mas o protocolo é conferido abaixo: sem isso,
 * `[clique](javascript:...)` viraria um link executável, que o DOMPurify
 * permite por padrão em alguns modos.
 */
const ATRIBUTOS_PERMITIDOS = ['href', 'title', 'colspan', 'rowspan']

/** Só links de verdade. Nada de javascript:, data:, vbscript:. */
const PROTOCOLOS_SEGUROS = /^(https?:|mailto:)/i

let configurado = false

function configurar() {
  if (configurado) return
  configurado = true

  /*
   * Links abrem em aba nova e sem acesso à janela de origem.
   *
   * `noopener` é o que importa: sem ele, a página aberta alcança
   * `window.opener` e pode redirecionar a aba do VisiON para uma tela de login
   * falsa — o usuário volta, vê o que parece o sistema, e digita a senha.
   */
  DOMPurify.addHook('afterSanitizeAttributes', node => {
    if (!(node instanceof Element) || node.tagName !== 'A') return

    const href = node.getAttribute('href') ?? ''
    if (!PROTOCOLOS_SEGUROS.test(href)) {
      node.removeAttribute('href')
      return
    }
    node.setAttribute('target', '_blank')
    node.setAttribute('rel', 'noopener noreferrer')
  })

  marked.setOptions({
    // Quebra de linha simples vira <br>: o modelo escreve como se conversasse,
    // e sem isto as linhas se juntam num parágrafo só.
    breaks: true,
    gfm: true,
  })
}

/**
 * Converte o Markdown do assistente em HTML seguro para `v-html`.
 *
 * Entrada vazia devolve string vazia — nunca `undefined`, que o Vue renderiza
 * como a palavra "undefined" na tela.
 */
export function renderizarMarkdown(texto: string): string {
  if (!texto) return ''
  configurar()

  const bruto = marked.parse(texto, { async: false }) as string

  return DOMPurify.sanitize(bruto, {
    ALLOWED_TAGS: TAGS_PERMITIDAS,
    ALLOWED_ATTR: ATRIBUTOS_PERMITIDOS,
    // target sobrevive ao hook acima; sem isto o DOMPurify o removeria depois.
    ADD_ATTR: ['target', 'rel'],
    // Nenhum protocolo exótico, nem em atributos que não sejam href.
    ALLOW_DATA_ATTR: false,
    ALLOW_UNKNOWN_PROTOCOLS: false,
  })
}
