package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"omie-sync-api/internal/apperror"
)

// repoContexto permite montar cada cenário de papel, vínculo e sessão.
type repoContexto struct {
	usuario   *Usuario
	grupos    []GrupoInfo
	temVinc   bool
	roleGrupo string
	rt        *RefreshToken
	// o que issueTokens chegou a persistir
	ctxGravado   Contexto
	grupoGravado string
}

func (r *repoContexto) GetUsuarioByEmail(_ context.Context, _ string) (*Usuario, error) {
	return r.usuario, nil
}

func (r *repoContexto) GetUsuarioByID(_ context.Context, _ string) (*Usuario, error) {
	return r.usuario, nil
}

func (r *repoContexto) InsertRefreshToken(_ context.Context, _, token string, exp time.Time, grupoID string, c Contexto) (*RefreshToken, error) {
	r.ctxGravado, r.grupoGravado = c, grupoID
	return &RefreshToken{Token: token, ExpiresAt: exp, GrupoID: grupoID, Contexto: c}, nil
}

func (r *repoContexto) UpdateSenhaPropria(_ context.Context, _, _ string) error { return nil }

func (r *repoContexto) GetRefreshToken(_ context.Context, _ string) (*RefreshToken, error) {
	if r.rt == nil {
		return nil, errors.New("não encontrado")
	}
	return r.rt, nil
}

func (r *repoContexto) RevokeRefreshToken(_ context.Context, _ string) error  { return nil }
func (r *repoContexto) RevokeAllUserTokens(_ context.Context, _ string) error { return nil }

func (r *repoContexto) GetGruposByUsuarioID(_ context.Context, _ string) ([]GrupoInfo, error) {
	return r.grupos, nil
}

func (r *repoContexto) ValidateUsuarioGrupo(_ context.Context, _, _ string) (bool, error) {
	return r.temVinc, nil
}

func (r *repoContexto) GetRoleNoGrupo(_ context.Context, _, _ string) (string, error) {
	return r.roleGrupo, nil
}

func usuarioCom(role string) *Usuario {
	return &Usuario{
		ID: "u-1", GrupoID: gA, Nome: "Teste", Email: "t@e.com",
		Password: hashedPassword("senha123"), Role: role, Ativo: true,
	}
}

func claimsDoToken(t *testing.T, token string) *JWTClaims {
	t.Helper()
	c, err := NewJWTService(testSecret).Validate(token)
	if err != nil {
		t.Fatalf("validar token emitido: %v", err)
	}
	return c
}

func preAuthDeTeste(t *testing.T) string {
	t.Helper()
	tok, err := NewJWTService(testSecret).GeneratePreAuth("u-1", "t@e.com")
	if err != nil {
		t.Fatalf("gerar pre-auth: %v", err)
	}
	return tok
}

/*
O caso que motivou a fase: o admin global com um grupo só entrava direto nesse
grupo, e o painel de plataforma dele ficava sem caminho de volta que não fosse
deslogar.
*/
func TestService_Login_AdminGlobalComUmGrupoEscolhe(t *testing.T) {
	repo := &repoContexto{usuario: usuarioCom(RolePlataforma), grupos: []GrupoInfo{{ID: gA}}, temVinc: true}
	resp, err := newTestService(repo).Login(context.Background(), "t@e.com", "senha123")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if !resp.NeedsSelect {
		t.Fatal("entraria direto no grupo e perderia a plataforma")
	}
	if !resp.PodePlataforma {
		t.Fatal("a tela não teria como oferecer Plataforma")
	}
	if resp.AccessToken != "" {
		t.Fatal("token emitido antes da escolha")
	}
}

// Quem não é admin global segue como antes: um grupo entra direto.
func TestService_Login_UmGrupoEntraDireto(t *testing.T) {
	repo := &repoContexto{
		usuario: usuarioCom(RoleAdminGrupo), grupos: []GrupoInfo{{ID: gA}},
		temVinc: true, roleGrupo: RoleAdminGrupo,
	}
	resp, err := newTestService(repo).Login(context.Background(), "t@e.com", "senha123")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if resp.NeedsSelect || resp.AccessToken == "" {
		t.Fatalf("got %+v", resp)
	}
	c := claimsDoToken(t, resp.AccessToken)
	if c.Contexto != ContextoGrupo || c.GrupoID != gA || c.Role != RoleAdminGrupo {
		t.Fatalf("claims: %+v", c)
	}
	// O contexto precisa sobreviver à renovação silenciosa, e ela só tem o
	// refresh token opaco para consultar.
	if repo.ctxGravado != ContextoGrupo {
		t.Fatalf("contexto não persistido no refresh: %q", repo.ctxGravado)
	}
	if resp.PodePlataforma {
		t.Fatal("admin de grupo não pode oferecer Plataforma")
	}
}

func TestService_SelectGrupo_Contexto(t *testing.T) {
	t.Run("admin global entra na plataforma", func(t *testing.T) {
		repo := &repoContexto{usuario: usuarioCom(RolePlataforma)}
		resp, err := newTestService(repo).SelectGrupo(context.Background(), preAuthDeTeste(t), ContextoPlataforma, "")
		if err != nil {
			t.Fatalf("%v", err)
		}
		c := claimsDoToken(t, resp.AccessToken)
		if c.Contexto != ContextoPlataforma || c.Role != RolePlataforma || c.GrupoID != "" {
			t.Fatalf("claims: %+v", c)
		}
		if repo.ctxGravado != ContextoPlataforma || repo.grupoGravado != "" {
			t.Fatalf("refresh gravado errado: %q / %q", repo.ctxGravado, repo.grupoGravado)
		}
	})

	// A trava que faz "existe um admin global" valer na emissão do token.
	t.Run("admin de grupo não entra na plataforma", func(t *testing.T) {
		repo := &repoContexto{usuario: usuarioCom(RoleAdminGrupo)}
		_, err := newTestService(repo).SelectGrupo(context.Background(), preAuthDeTeste(t), ContextoPlataforma, "")
		if ae, ok := apperror.IsAppError(err); !ok || ae.Code != 403 {
			t.Fatalf("esperava 403, got %v", err)
		}
	})

	// O poder de plataforma não viaja para dentro de um cliente.
	t.Run("admin global dentro de um grupo vira admin daquele grupo", func(t *testing.T) {
		repo := &repoContexto{usuario: usuarioCom(RolePlataforma), temVinc: true, roleGrupo: RoleAdminGrupo}
		resp, err := newTestService(repo).SelectGrupo(context.Background(), preAuthDeTeste(t), ContextoGrupo, gA)
		if err != nil {
			t.Fatalf("%v", err)
		}
		c := claimsDoToken(t, resp.AccessToken)
		if c.Role != RoleAdminGrupo || c.Contexto != ContextoGrupo {
			t.Fatalf("claims: %+v", c)
		}
	})

	t.Run("grupo sem vínculo é recusado", func(t *testing.T) {
		repo := &repoContexto{usuario: usuarioCom(RoleAdminGrupo), temVinc: false}
		_, err := newTestService(repo).SelectGrupo(context.Background(), preAuthDeTeste(t), ContextoGrupo, gA)
		if ae, ok := apperror.IsAppError(err); !ok || ae.Code != 403 {
			t.Fatalf("esperava 403, got %v", err)
		}
	})
}

// A troca de contexto é a mesma regra da entrada — não uma segunda cópia dela.
func TestService_TrocaGrupo_UsaAMesmaRegra(t *testing.T) {
	repo := &repoContexto{usuario: usuarioCom(RoleAdminGrupo), temVinc: true, roleGrupo: RoleAdminGrupo}
	_, err := newTestService(repo).TrocaGrupo(context.Background(), "u-1", ContextoPlataforma, "")
	if ae, ok := apperror.IsAppError(err); !ok || ae.Code != 403 {
		t.Fatalf("admin de grupo assumiu a plataforma pela troca: %v", err)
	}
}

/*
A renovação revalida, em vez de reemitir o que estava.

Sem isso, rebaixar alguém — ou tirá-lo de um grupo — só surtia efeito quando o
refresh token expirasse, até sete dias depois: a renovação reemitia o papel
antigo indefinidamente.
*/
func TestService_Refresh_Revalida(t *testing.T) {
	rt := func(c Contexto, g string) *RefreshToken {
		return &RefreshToken{UsuarioID: "u-1", Token: "tok", GrupoID: g, Contexto: c, ExpiresAt: time.Now().Add(time.Hour)}
	}

	t.Run("preserva o contexto da sessão", func(t *testing.T) {
		repo := &repoContexto{usuario: usuarioCom(RolePlataforma), rt: rt(ContextoPlataforma, "")}
		resp, err := newTestService(repo).Refresh(context.Background(), "tok")
		if err != nil {
			t.Fatalf("%v", err)
		}
		if c := claimsDoToken(t, resp.AccessToken); c.Contexto != ContextoPlataforma {
			t.Fatalf("a renovação trocou o contexto sozinha: %+v", c)
		}
	})

	t.Run("rebaixamento derruba a sessão de plataforma", func(t *testing.T) {
		repo := &repoContexto{usuario: usuarioCom(RoleAdminGrupo), rt: rt(ContextoPlataforma, "")}
		_, err := newTestService(repo).Refresh(context.Background(), "tok")
		if ae, ok := apperror.IsAppError(err); !ok || ae.Code != 403 {
			t.Fatalf("papel antigo reemitido: %v", err)
		}
	})

	t.Run("vínculo removido derruba a sessão de grupo", func(t *testing.T) {
		repo := &repoContexto{usuario: usuarioCom(RoleAdminGrupo), temVinc: false, rt: rt(ContextoGrupo, gA)}
		_, err := newTestService(repo).Refresh(context.Background(), "tok")
		if ae, ok := apperror.IsAppError(err); !ok || ae.Code != 403 {
			t.Fatalf("sessão sobreviveu à remoção do grupo: %v", err)
		}
	})

	/*
	 * Token anterior à migration 000031, que revogou todos — mas se algum
	 * escapar, contexto vazio é lido como grupo. Omissão não pode promover
	 * ninguém à plataforma.
	 */
	t.Run("sessão sem contexto não vira plataforma", func(t *testing.T) {
		repo := &repoContexto{usuario: usuarioCom(RolePlataforma), temVinc: true, roleGrupo: RoleAdminGrupo, rt: rt("", gA)}
		resp, err := newTestService(repo).Refresh(context.Background(), "tok")
		if err != nil {
			t.Fatalf("%v", err)
		}
		c := claimsDoToken(t, resp.AccessToken)
		if c.Contexto != ContextoGrupo || c.Role == RolePlataforma {
			t.Fatalf("omissão promoveu à plataforma: %+v", c)
		}
	})
}

func TestService_GetContextos(t *testing.T) {
	grupos := []GrupoInfo{{ID: gA, Nome: "Alpha"}}

	repo := &repoContexto{usuario: usuarioCom(RolePlataforma), grupos: grupos}
	c, err := newTestService(repo).GetContextos(context.Background(), "u-1")
	if err != nil || !c.PodePlataforma || len(c.Grupos) != 1 {
		t.Fatalf("got %+v, %v", c, err)
	}

	repo = &repoContexto{usuario: usuarioCom(RoleAdminGrupo), grupos: grupos}
	c, _ = newTestService(repo).GetContextos(context.Background(), "u-1")
	if c.PodePlataforma {
		t.Fatal("admin de grupo não pode ver Plataforma na lista")
	}
}

/*
Token sem a claim de contexto é recusado.

É a metade que faz a troca valer de imediato, em vez de conviver com dois
formatos: sem isso, todo access token emitido antes da 000031 continuaria aceito
por até quinze minutos, e — pior — qualquer token futuro sem a claim cairia num
contexto implícito.
*/
func TestJWT_TokenSemContextoERecusado(t *testing.T) {
	svc := NewJWTService(testSecret)

	semCtx, err := svc.Generate("u-1", gA, "t@e.com", RoleViewer, Contexto(""), false)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if _, err := svc.Validate(semCtx); err == nil {
		t.Fatal("token sem contexto foi aceito")
	}

	invalido, _ := svc.Generate("u-1", gA, "t@e.com", RoleViewer, Contexto("qualquer"), false)
	if _, err := svc.Validate(invalido); err == nil {
		t.Fatal("contexto desconhecido foi aceito")
	}

	for _, c := range []Contexto{ContextoPlataforma, ContextoGrupo} {
		tok, _ := svc.Generate("u-1", gA, "t@e.com", RoleViewer, c, false)
		claims, err := svc.Validate(tok)
		if err != nil {
			t.Fatalf("contexto %q deveria ser aceito: %v", c, err)
		}
		if claims.Contexto != c {
			t.Fatalf("contexto perdido na ida e volta: %q", claims.Contexto)
		}
	}
}
