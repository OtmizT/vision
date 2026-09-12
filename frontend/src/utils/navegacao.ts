/**
 * Para onde cada papel vai ao entrar, e o que ele vê no menu.
 *
 * O admin global deixou de ter Dashboard: ele administra a plataforma e não
 * consome os dados financeiros de nenhum grupo. Isso quebrou a premissa que
 * estava espalhada pelo código de que `/` serve para todo mundo — o redirect
 * pós-login aparece em três lugares (o guard da rota pública, a tela de login e
 * a de seleção de grupo), e os três mandavam para `/`.
 *
 * Puro e testado porque um erro aqui não dá erro: manda o usuário para uma tela
 * que ele não pode ver, e ele recebe um 403 sem entender por quê.
 */

export type Papel = 'admin_global' | 'admin_grupo' | 'viewer'

/** Onde o admin global trabalha: a visão de sync de todas as empresas. */
export const ROTA_ADMIN_GLOBAL = '/admin/sync-control'

/** Dashboard, a tela principal do produto para quem consome dados. */
export const ROTA_PADRAO = '/'

export function rotaInicial(papel: Papel | undefined | null): string {
  return papel === 'admin_global' ? ROTA_ADMIN_GLOBAL : ROTA_PADRAO
}

/**
 * O Dashboard aparece para este papel?
 *
 * Uma função, e não um `!==` solto no menu e outro no roteador: são dois lugares
 * que precisam concordar, e discordando o item some do menu mas a rota continua
 * acessível pela URL — ou o contrário, que é pior.
 */
export function temDashboard(papel: Papel | undefined | null): boolean {
  return papel !== 'admin_global'
}

/**
 * Destino seguro depois de entrar, respeitando um `?redirect=` da URL.
 *
 * O redirect é honrado apenas quando é um caminho interno: sem isso, um link
 * com `?redirect=https://outro.site` levaria o usuário recém-autenticado para
 * fora. Caminho protocolo-relativo (`//`) também sai, porque o navegador o trata
 * como externo.
 */
export function destinoAposEntrar(papel: Papel | undefined | null, redirect?: string | null): string {
  if (redirect && redirect.startsWith('/') && !redirect.startsWith('//')) return redirect
  return rotaInicial(papel)
}
