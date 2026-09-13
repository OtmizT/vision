package auth

import "omie-sync-api/internal/apperror"

/*
Onde a pessoa está trabalhando, e o que ela pode ali.

Este arquivo não fala com banco nem com HTTP de propósito. A regra de
autorização mais importante do sistema — quem é admin da plataforma e quem é
admin de um cliente — vivia como um CASE WHEN dentro de uma string SQL em
auth/repository.go, sem um único teste, e o bloco que consumia o resultado
estava copiado em quatro lugares de service.go (Login, SelectGrupo, TrocaGrupo e
Refresh). Quatro cópias de uma regra é quatro chances de ela divergir, e a
divergência aqui não dá erro: dá acesso a mais ou a menos do que devia.

Contexto é a ideia que faltava. Hoje o sistema só sabe "que grupo está ativo", e
por isso o admin global fica preso fora das telas de grupo: o CASE WHEN faz
admin_global vencer o papel de grupo dele sempre. Separar plataforma de grupo é
o que permite a mesma pessoa administrar a plataforma numa hora e olhar os dados
de um cliente na outra, sem que um papel contamine o outro.
*/
type Contexto string

const (
	// ContextoPlataforma administra o produto: grupos, sync de todos os
	// clientes, manutenção. Não olha dado financeiro de ninguém.
	ContextoPlataforma Contexto = "plataforma"
	// ContextoGrupo é o trabalho dentro de um cliente.
	ContextoGrupo Contexto = "grupo"
)

// RolePlataforma é o papel global — mora em usuarios.role, e é o único lugar de
// onde o poder de plataforma pode vir.
const RolePlataforma = "admin_global"

const (
	RoleAdminGrupo = "admin_grupo"
	RoleViewer     = "viewer"
)

/*
RoleEfetiva decide com que papel o token é emitido.

Os parâmetros são todos dados já lidos pelo chamador, nunca I/O: é o que torna a
tabela-verdade inteira testável sem banco.

  - roleGlobal    usuarios.role, o papel de plataforma
  - roleNoGrupo   usuario_grupos.role, ou "" se não houver vínculo/coluna
  - temVinculo    se existe linha em usuario_grupos para este par

No contexto de grupo o papel vem do vínculo, com queda para o papel global
quando o vínculo não diz nada — que é o comportamento que as quatro cópias já
tinham, para bancos com a migration 000024 pendente.
*/
func RoleEfetiva(ctx Contexto, grupoID, roleGlobal, roleNoGrupo string, temVinculo bool) (string, error) {
	switch ctx {
	case ContextoPlataforma:
		if grupoID != "" {
			// Plataforma com grupo preenchido é estado impossível: ou se
			// administra o produto, ou se está dentro de um cliente. Aceitar
			// os dois ao mesmo tempo é como o isolamento se perde sem ninguém
			// perceber.
			return "", apperror.Forbidden("contexto de plataforma não tem grupo")
		}
		if roleGlobal != RolePlataforma {
			return "", apperror.Forbidden("apenas o admin global assume o contexto de plataforma")
		}
		return RolePlataforma, nil

	case ContextoGrupo:
		if grupoID == "" {
			return "", apperror.Forbidden("contexto de grupo exige um grupo")
		}
		if !temVinculo {
			return "", apperror.Forbidden("usuário não pertence a este grupo")
		}

		/*
		 * admin_global gravado no vínculo não concede nada.
		 *
		 * A migration 000030 impede que esse valor exista, mas a defesa fica
		 * aqui também: a coluna passou seis migrations sem CHECK, e a 000025
		 * copiou usuarios.role para dentro dela em massa.
		 *
		 * O plano previa devolver erro neste caso. Erro tranca: se a migration
		 * não tiver rodado ainda — a janela de um deploy — o admin global não
		 * entraria em lugar nenhum, e não há um segundo administrador para
		 * socorrer. Ignorar o valor e cair no papel global neutraliza a
		 * escalada sem esse risco: quem é admin_global de verdade continua
		 * sendo, porque isso está em usuarios.role; quem não é recebe o papel
		 * que realmente tem.
		 */
		if roleNoGrupo != "" && roleNoGrupo != RolePlataforma {
			return roleNoGrupo, nil
		}
		if roleGlobal == RolePlataforma {
			// O admin global dentro de um grupo trabalha como admin daquele
			// grupo. O poder de plataforma dele não viaja para cá.
			return RoleAdminGrupo, nil
		}
		if roleGlobal != "" {
			return roleGlobal, nil
		}
		return RoleViewer, nil
	}

	return "", apperror.Forbidden("contexto inválido")
}

/*
PodeAssumir responde se a pessoa pode entrar neste contexto, sem se importar com
qual papel ela teria. É a pergunta que a troca de contexto faz.
*/
func PodeAssumir(ctx Contexto, grupoID, roleGlobal, roleNoGrupo string, temVinculo bool) error {
	_, err := RoleEfetiva(ctx, grupoID, roleGlobal, roleNoGrupo, temVinculo)
	return err
}

// DecisaoLogin diz o que fazer logo depois de conferir a senha.
type DecisaoLogin struct {
	// PedirSelecao: a pessoa escolhe onde entrar antes de receber o token.
	PedirSelecao bool
	// Contexto e GrupoID valem quando PedirSelecao é false.
	Contexto Contexto
	GrupoID  string
}

/*
DecidirEntrada escolhe entre entrar direto e pedir seleção.

O caso que o plano marcou e que o código de hoje erra: **admin global com um
grupo só**. A condição atual é `len(grupos) > 1`, então ele entraria direto
naquele grupo — e a plataforma, que é onde ele trabalha, não teria caminho de
volta a não ser deslogar. Para quem é admin global a seleção é sempre oferecida,
porque Plataforma é sempre uma das opções.

Quem não é admin global segue a regra antiga: um grupo entra direto, vários
pedem escolha, nenhum entra no grupo do cadastro (o fallback legado).
*/
func DecidirEntrada(roleGlobal string, grupos []GrupoInfo, grupoLegado string) DecisaoLogin {
	if roleGlobal == RolePlataforma {
		if len(grupos) == 0 {
			// Sem grupo nenhum não há o que escolher: só existe a plataforma.
			return DecisaoLogin{Contexto: ContextoPlataforma}
		}
		return DecisaoLogin{PedirSelecao: true}
	}

	if len(grupos) > 1 {
		return DecisaoLogin{PedirSelecao: true}
	}
	if len(grupos) == 1 {
		return DecisaoLogin{Contexto: ContextoGrupo, GrupoID: grupos[0].ID}
	}
	return DecisaoLogin{Contexto: ContextoGrupo, GrupoID: grupoLegado}
}
