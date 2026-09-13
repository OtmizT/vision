/*
Papéis: o que é de plataforma e o que é de grupo.

A tela de usuários vive em /admin/grupos/{grupoID}/usuarios e oferecia "Admin
Global" no seletor quando quem editava já era admin global. Isso concedia, por
uma rota de grupo, o papel que manda na plataforma inteira — e o backend
aceitava, gravando na coluna global `usuarios.role`.

O backend deixou de aceitar (internal/usuarios/types.go) e o banco passou a
recusar no vínculo (migration 000030). Aqui a opção sai do seletor, para que
ninguém escolha algo que vai voltar 422.

admin_global continua sendo EXIBIDO: a conta existe, aparece na lista do grupo a
que está vinculada, e esconder o papel dela faria a lista mentir. O que muda é
que não se atribui mais por aqui.
*/

export type PapelDeGrupo = 'admin_grupo' | 'viewer'
export type Papel = PapelDeGrupo | 'admin_global'

export interface OpcaoPapel {
  value: PapelDeGrupo
  label: string
}

/** As únicas opções atribuíveis pela tela de usuários de um grupo. */
export function papeisAtribuiveis(): OpcaoPapel[] {
  return [
    { value: 'admin_grupo', label: 'Admin Grupo' },
    { value: 'viewer', label: 'Viewer' },
  ]
}

/** Rótulo de exibição, inclusive para o papel que não se atribui mais. */
export function rotuloPapel(p: string): string {
  switch (p) {
    case 'admin_global':
      return 'Admin Global'
    case 'admin_grupo':
      return 'Admin Grupo'
    case 'viewer':
      return 'Viewer'
    default:
      return p
  }
}
