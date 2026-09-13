package auth

import (
	"testing"

	"omie-sync-api/internal/apperror"
)

const gA = "11111111-1111-1111-1111-111111111111"

func negado(t *testing.T, err error) {
	t.Helper()
	ae, ok := apperror.IsAppError(err)
	if !ok || ae.Code != 403 {
		t.Fatalf("esperava 403, got %v", err)
	}
}

func TestRoleEfetiva_Plataforma(t *testing.T) {
	t.Run("admin global assume", func(t *testing.T) {
		r, err := RoleEfetiva(ContextoPlataforma, "", RolePlataforma, "", false)
		if err != nil || r != RolePlataforma {
			t.Fatalf("got %q, %v", r, err)
		}
	})

	// A regra que faz "existe um admin global" valer na emissão do token.
	t.Run("admin de grupo não assume a plataforma", func(t *testing.T) {
		_, err := RoleEfetiva(ContextoPlataforma, "", RoleAdminGrupo, "", false)
		negado(t, err)
	})

	t.Run("viewer não assume a plataforma", func(t *testing.T) {
		_, err := RoleEfetiva(ContextoPlataforma, "", RoleViewer, "", false)
		negado(t, err)
	})

	// Papel de grupo nunca promove a plataforma, venha ele de onde vier.
	t.Run("vínculo dizendo admin_global não promove", func(t *testing.T) {
		_, err := RoleEfetiva(ContextoPlataforma, "", RoleViewer, RolePlataforma, true)
		negado(t, err)
	})

	/*
	 * Plataforma com grupo preenchido é estado impossível: ou se administra o
	 * produto, ou se está dentro de um cliente. Aceitar os dois ao mesmo tempo
	 * é como o isolamento se perderia sem ninguém perceber.
	 */
	t.Run("plataforma com grupo preenchido é recusada", func(t *testing.T) {
		_, err := RoleEfetiva(ContextoPlataforma, gA, RolePlataforma, "", true)
		negado(t, err)
	})
}

func TestRoleEfetiva_Grupo(t *testing.T) {
	t.Run("o papel vem do vínculo", func(t *testing.T) {
		r, err := RoleEfetiva(ContextoGrupo, gA, RoleViewer, RoleAdminGrupo, true)
		if err != nil || r != RoleAdminGrupo {
			t.Fatalf("got %q, %v", r, err)
		}
	})

	t.Run("viewer no grupo é viewer, mesmo sendo admin de outro", func(t *testing.T) {
		r, _ := RoleEfetiva(ContextoGrupo, gA, RoleAdminGrupo, RoleViewer, true)
		if r != RoleViewer {
			t.Fatalf("got %q, want viewer", r)
		}
	})

	t.Run("sem vínculo não entra", func(t *testing.T) {
		_, err := RoleEfetiva(ContextoGrupo, gA, RoleAdminGrupo, RoleAdminGrupo, false)
		negado(t, err)
	})

	t.Run("grupo vazio é recusado", func(t *testing.T) {
		_, err := RoleEfetiva(ContextoGrupo, "", RoleAdminGrupo, RoleAdminGrupo, true)
		negado(t, err)
	})

	/*
	 * A defesa em profundidade da migration 000030: a coluna passou seis
	 * migrations sem CHECK, e a 000025 copiou usuarios.role para dentro dela.
	 * O valor é ignorado em vez de virar erro — errar aqui trancaria o único
	 * admin global fora do sistema na janela de um deploy, sem ninguém para
	 * socorrer.
	 */
	t.Run("vínculo dizendo admin_global não concede plataforma", func(t *testing.T) {
		r, err := RoleEfetiva(ContextoGrupo, gA, RoleViewer, RolePlataforma, true)
		if err != nil {
			t.Fatalf("não deveria trancar: %v", err)
		}
		if r == RolePlataforma {
			t.Fatal("escalada: papel de grupo virou papel de plataforma")
		}
		if r != RoleViewer {
			t.Fatalf("got %q, want viewer (o papel que a pessoa realmente tem)", r)
		}
	})

	// O poder de plataforma não viaja para dentro de um cliente.
	t.Run("admin global dentro de um grupo é admin daquele grupo", func(t *testing.T) {
		r, err := RoleEfetiva(ContextoGrupo, gA, RolePlataforma, "", true)
		if err != nil {
			t.Fatalf("%v", err)
		}
		if r != RoleAdminGrupo {
			t.Fatalf("got %q, want admin_grupo", r)
		}
	})

	// Banco com a migration 000024 pendente: a coluna role não existe.
	t.Run("sem papel no vínculo cai no papel global", func(t *testing.T) {
		r, _ := RoleEfetiva(ContextoGrupo, gA, RoleAdminGrupo, "", true)
		if r != RoleAdminGrupo {
			t.Fatalf("got %q, want admin_grupo", r)
		}
	})

	// Nunca devolver "" — um papel vazio não casa com nenhum RequireRole e
	// produziria 403 sem explicação em toda tela.
	t.Run("sem papel nenhum cai em viewer", func(t *testing.T) {
		r, err := RoleEfetiva(ContextoGrupo, gA, "", "", true)
		if err != nil || r != RoleViewer {
			t.Fatalf("got %q, %v", r, err)
		}
	})
}

func TestRoleEfetiva_ContextoInvalido(t *testing.T) {
	_, err := RoleEfetiva(Contexto("qualquer"), gA, RolePlataforma, RoleAdminGrupo, true)
	negado(t, err)
}

// Nenhum caminho pode devolver papel vazio junto com erro nulo.
func TestRoleEfetiva_NuncaDevolvePapelVazioSemErro(t *testing.T) {
	ctxs := []Contexto{ContextoPlataforma, ContextoGrupo, Contexto("x")}
	papeis := []string{"", RoleViewer, RoleAdminGrupo, RolePlataforma}
	grupos := []string{"", gA}

	for _, c := range ctxs {
		for _, rg := range papeis {
			for _, rn := range papeis {
				for _, g := range grupos {
					for _, v := range []bool{true, false} {
						r, err := RoleEfetiva(c, g, rg, rn, v)
						if err == nil && r == "" {
							t.Fatalf("papel vazio sem erro: ctx=%q grupo=%q global=%q vinculo=%q temVinculo=%v", c, g, rg, rn, v)
						}
						if err == nil && r == RolePlataforma && c != ContextoPlataforma {
							t.Fatalf("papel de plataforma fora da plataforma: ctx=%q global=%q vinculo=%q", c, rg, rn)
						}
					}
				}
			}
		}
	}
}

func TestPodeAssumir(t *testing.T) {
	if err := PodeAssumir(ContextoPlataforma, "", RolePlataforma, "", false); err != nil {
		t.Fatalf("admin global deveria assumir a plataforma: %v", err)
	}
	// O caso do plano: admin de grupo tentando assumir Plataforma pela API.
	negado(t, PodeAssumir(ContextoPlataforma, "", RoleAdminGrupo, "", false))
	negado(t, PodeAssumir(ContextoGrupo, gA, RoleAdminGrupo, RoleAdminGrupo, false))
}

func TestDecidirEntrada(t *testing.T) {
	umGrupo := []GrupoInfo{{ID: gA, Nome: "Alpha"}}
	doisGrupos := []GrupoInfo{{ID: gA}, {ID: "22222222-2222-2222-2222-222222222222"}}

	/*
	 * O caso que o código de hoje erra. A condição é `len(grupos) > 1`, então o
	 * admin global com um grupo só entraria direto naquele grupo — e a
	 * plataforma, que é onde ele trabalha, não teria caminho de volta a não ser
	 * deslogar.
	 */
	t.Run("admin global com um grupo ainda escolhe", func(t *testing.T) {
		if d := DecidirEntrada(RolePlataforma, umGrupo, ""); !d.PedirSelecao {
			t.Fatalf("entraria direto no grupo e perderia a plataforma: %+v", d)
		}
	})

	t.Run("admin global sem grupo nenhum vai para a plataforma", func(t *testing.T) {
		d := DecidirEntrada(RolePlataforma, nil, "")
		if d.PedirSelecao || d.Contexto != ContextoPlataforma {
			t.Fatalf("got %+v", d)
		}
	})

	t.Run("um grupo entra direto", func(t *testing.T) {
		d := DecidirEntrada(RoleAdminGrupo, umGrupo, "")
		if d.PedirSelecao || d.Contexto != ContextoGrupo || d.GrupoID != gA {
			t.Fatalf("got %+v", d)
		}
	})

	t.Run("vários grupos pedem escolha", func(t *testing.T) {
		if d := DecidirEntrada(RoleViewer, doisGrupos, ""); !d.PedirSelecao {
			t.Fatalf("got %+v", d)
		}
	})

	// Banco com a junction pendente: o grupo vem de usuarios.grupo_id.
	t.Run("sem grupos usa o grupo do cadastro", func(t *testing.T) {
		d := DecidirEntrada(RoleViewer, nil, gA)
		if d.PedirSelecao || d.GrupoID != gA {
			t.Fatalf("got %+v", d)
		}
	})

	// Pedir seleção e já dizer o destino ao mesmo tempo é contraditório.
	t.Run("quem pede seleção não traz destino", func(t *testing.T) {
		d := DecidirEntrada(RoleViewer, doisGrupos, gA)
		if d.GrupoID != "" || d.Contexto != "" {
			t.Fatalf("got %+v", d)
		}
	})
}
