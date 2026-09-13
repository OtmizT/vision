/*
Para onde cada contexto vai ao entrar, e o que ele vê no menu.

Antes isto decidia pelo **papel**: admin_global ia para o Sync Control, o resto
para o Dashboard. Era a aproximação possível enquanto "admin global" e "onde a
pessoa está" eram a mesma coisa — e o efeito era que o admin global ficava preso
fora das telas de grupo, sem poder olhar os dados de nenhum cliente.

Agora decide pelo **contexto**. A mesma pessoa administra a plataforma numa hora
e entra num cliente na outra, e o menu acompanha: Plataforma não tem Dashboard,
Grupo não tem Sync Control.

Puro e testado porque um erro aqui não dá erro: manda o usuário para uma tela
que ele não pode ver, e ele recebe um 403 sem entender por quê.
*/

export type Papel = 'admin_global' | 'admin_grupo' | 'viewer'
export type Contexto = 'plataforma' | 'grupo'

/** Onde o contexto de plataforma trabalha: o sync de todas as empresas. */
export const ROTA_PLATAFORMA = '/admin/sync-control'

/** Dashboard, a tela principal de quem consome dados de um cliente. */
export const ROTA_PADRAO = '/'

/**
 * Contexto vazio é lido como 'grupo'.
 *
 * O mesmo padrão do servidor (contextoOuGrupo, em auth/handler.go): omissão
 * nunca concede a plataforma. Aparece de verdade entre o login e o primeiro
 * /auth/me, quando `user` ainda é null.
 */
export function contextoOuGrupo(c: Contexto | string | undefined | null): Contexto {
  return c === 'plataforma' ? 'plataforma' : 'grupo'
}

export function rotaInicial(contexto: Contexto | string | undefined | null): string {
  return contextoOuGrupo(contexto) === 'plataforma' ? ROTA_PLATAFORMA : ROTA_PADRAO
}

/**
 * O Dashboard aparece neste contexto?
 *
 * Uma função, e não um `!==` solto no menu e outro no roteador: são dois
 * lugares que precisam concordar, e discordando o item some do menu mas a rota
 * continua acessível pela URL — ou o contrário, que é pior.
 */
export function temDashboard(contexto: Contexto | string | undefined | null): boolean {
  return contextoOuGrupo(contexto) === 'grupo'
}

/** O Sync Control é o inverso: só existe na plataforma. */
export function temSyncControl(contexto: Contexto | string | undefined | null): boolean {
  return contextoOuGrupo(contexto) === 'plataforma'
}

/**
 * Rótulo do badge que diz onde a pessoa está.
 *
 * Sem ele o admin global dentro de um grupo se lê como "Admin Grupo" na tela,
 * sem nenhuma pista de que é a mesma conta que administra a plataforma.
 */
export function rotuloContexto(contexto: Contexto | string | undefined | null, nomeDoGrupo?: string): string {
  if (contextoOuGrupo(contexto) === 'plataforma') return 'Plataforma'
  return nomeDoGrupo?.trim() || 'Grupo'
}

/**
 * Destino seguro depois de entrar, respeitando um `?redirect=` da URL.
 *
 * O redirect é honrado apenas quando é um caminho interno: sem isso, um link
 * com `?redirect=https://outro.site` levaria o usuário recém-autenticado para
 * fora. Caminho protocolo-relativo (`//`) também sai, porque o navegador o
 * trata como externo.
 */
export function destinoAposEntrar(
  contexto: Contexto | string | undefined | null,
  redirect?: string | null,
): string {
  if (redirect && redirect.startsWith('/') && !redirect.startsWith('//')) return redirect
  return rotaInicial(contexto)
}
