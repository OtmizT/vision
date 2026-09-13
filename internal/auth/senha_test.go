package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"omie-sync-api/internal/apperror"
)

// repoSenha registra o que foi gravado e revogado.
type repoSenha struct {
	repoContexto
	hashGravado string
	revogou     bool
}

func (r *repoSenha) UpdateSenhaPropria(_ context.Context, _, hash string) error {
	r.hashGravado = hash
	return nil
}

func (r *repoSenha) RevokeAllUserTokens(_ context.Context, _ string) error {
	r.revogou = true
	return nil
}

func novoRepoSenha(senhaAtual string, provisoria bool) *repoSenha {
	u := usuarioCom(RoleViewer)
	u.Password = hashedPassword(senhaAtual)
	u.SenhaProvisoria = provisoria
	return &repoSenha{repoContexto: repoContexto{usuario: u, temVinc: true, roleGrupo: RoleViewer}}
}

func erroCom(t *testing.T, err error, code int) {
	t.Helper()
	ae, ok := apperror.IsAppError(err)
	if !ok || ae.Code != code {
		t.Fatalf("esperava %d, got %v", code, err)
	}
}

/*
O defeito que motivou o endpoint: a tela de Perfil chamava o endpoint
ADMINISTRATIVO, que exige papel de admin — e por isso um viewer não conseguia
trocar a própria senha, recebia 403 numa tela feita para ele.
*/
func TestTrocarSenhaPropria_ViewerTrocaAPropriaSenha(t *testing.T) {
	repo := novoRepoSenha("senha-antiga-1", false)
	svc := newTestService(repo)

	resp, err := svc.TrocarSenhaPropria(context.Background(), "u-1", ContextoGrupo, gA,
		TrocaSenhaRequest{SenhaAtual: "senha-antiga-1", SenhaNova: "senha-nova-9876"})
	if err != nil {
		t.Fatalf("viewer não conseguiu trocar a própria senha: %v", err)
	}

	if bcrypt.CompareHashAndPassword([]byte(repo.hashGravado), []byte("senha-nova-9876")) != nil {
		t.Fatal("a senha gravada não confere com a nova")
	}
	// A senha nova não pode ir para o banco em claro.
	if repo.hashGravado == "senha-nova-9876" {
		t.Fatal("senha gravada sem hash")
	}
	if !repo.revogou {
		t.Fatal("sessões antigas não foram derrubadas")
	}
	// Devolver token novo evita que a pessoa troque a senha e seja deslogada
	// em seguida pela própria troca.
	if resp.AccessToken == "" {
		t.Fatal("não devolveu sessão nova")
	}
	if c := claimsDoToken(t, resp.AccessToken); c.SenhaProvisoria {
		t.Fatal("a marca de provisória sobreviveu à troca")
	}
}

func TestTrocarSenhaPropria_ExigeASenhaAtual(t *testing.T) {
	// Sem esta prova, um token roubado bastaria para tomar a conta.
	t.Run("senha atual errada é recusada", func(t *testing.T) {
		repo := novoRepoSenha("senha-antiga-1", false)
		_, err := newTestService(repo).TrocarSenhaPropria(context.Background(), "u-1", ContextoGrupo, gA,
			TrocaSenhaRequest{SenhaAtual: "chute-errado", SenhaNova: "senha-nova-9876"})
		erroCom(t, err, 401)
		if repo.hashGravado != "" {
			t.Fatal("gravou a senha mesmo assim")
		}
	})

	/*
	 * Também quando a atual é provisória: quem acabou de entrar com ela a
	 * conhece, e abrir exceção aqui criaria um caminho de troca sem prova.
	 */
	t.Run("senha provisória não dispensa a prova", func(t *testing.T) {
		repo := novoRepoSenha("provisoria-123", true)
		_, err := newTestService(repo).TrocarSenhaPropria(context.Background(), "u-1", ContextoGrupo, gA,
			TrocaSenhaRequest{SenhaAtual: "", SenhaNova: "senha-nova-9876"})
		erroCom(t, err, 401)
	})
}

func TestTrocarSenhaPropria_Validacoes(t *testing.T) {
	casos := []struct {
		nome string
		req  TrocaSenhaRequest
		code int
	}{
		{"curta demais", TrocaSenhaRequest{SenhaAtual: "senha-antiga-1", SenhaNova: "abc"}, 422},
		// Trocar a senha por ela mesma nao troca nada, e deixaria de provisoria
		// uma senha que outra pessoa continua conhecendo.
		{"igual à atual", TrocaSenhaRequest{SenhaAtual: "senha-antiga-1", SenhaNova: "senha-antiga-1"}, 422},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			repo := novoRepoSenha("senha-antiga-1", false)
			_, err := newTestService(repo).TrocarSenhaPropria(context.Background(), "u-1", ContextoGrupo, gA, c.req)
			erroCom(t, err, c.code)
			if repo.hashGravado != "" {
				t.Fatal("gravou apesar da validação")
			}
		})
	}
}

// Trocar a senha não é motivo para mudar onde a pessoa está.
func TestTrocarSenhaPropria_PreservaOContexto(t *testing.T) {
	repo := novoRepoSenha("senha-antiga-1", true)
	repo.usuario.Role = RolePlataforma

	resp, err := newTestService(repo).TrocarSenhaPropria(context.Background(), "u-1", ContextoPlataforma, "",
		TrocaSenhaRequest{SenhaAtual: "senha-antiga-1", SenhaNova: "senha-nova-9876"})
	if err != nil {
		t.Fatalf("%v", err)
	}
	c := claimsDoToken(t, resp.AccessToken)
	if c.Contexto != ContextoPlataforma || c.Role != RolePlataforma {
		t.Fatalf("a troca de senha mudou o contexto: %+v", c)
	}
}

/*
A trava: enquanto a senha for provisória, nada além da troca funciona.

Obrigar a troca só na tela seria teatro — bastaria chamar a API direto para
seguir usando a conta com a senha que outra pessoa conhece.
*/
func TestSenhaProvisoria_BloqueiaTudoMenosATroca(t *testing.T) {
	jwtSvc := NewJWTService(testSecret)
	tok, err := jwtSvc.Generate("u-1", gA, "t@e.com", RoleViewer, ContextoGrupo, true)
	if err != nil {
		t.Fatalf("%v", err)
	}

	roda := func(path string) int {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		rec := httptest.NewRecorder()
		RequireAuth(jwtSvc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(rec, req)
		return rec.Code
	}

	// O caminho da troca, e o mínimo para percorrê-lo.
	for _, p := range []string{"/auth/senha", "/auth/me", "/auth/logout", "/auth/refresh"} {
		if got := roda(p); got != http.StatusOK {
			t.Errorf("%s deveria passar: got %d", p, got)
		}
	}

	// Todo o resto.
	for _, p := range []string{"/dados/fluxo-caixa", "/admin/sync/empresas", "/auth/grupos", "/auth/contextos", "/sync/1/stream"} {
		if got := roda(p); got != http.StatusForbidden {
			t.Errorf("%s deveria ser bloqueado: got %d", p, got)
		}
	}
}

// Sem a marca, nada muda para ninguém.
func TestSenhaProvisoria_NaoBloqueiaQuemNaoTem(t *testing.T) {
	jwtSvc := NewJWTService(testSecret)
	tok, _ := jwtSvc.Generate("u-1", gA, "t@e.com", RoleViewer, ContextoGrupo, false)

	req := httptest.NewRequest(http.MethodGet, "/dados/fluxo-caixa", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	RequireAuth(jwtSvc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rec.Code)
	}
}

// Um stream aberto é acesso como qualquer outro, e ficaria fora da trava se a
// checagem morasse só no RequireAuth comum.
func TestSenhaProvisoria_BloqueiaTambemOSSE(t *testing.T) {
	jwtSvc := NewJWTService(testSecret)
	tok, _ := jwtSvc.Generate("u-1", gA, "t@e.com", RoleViewer, ContextoGrupo, true)

	req := httptest.NewRequest(http.MethodGet, "/sync/1/stream?token="+tok, nil)
	rec := httptest.NewRecorder()
	RequireAuthSSE(jwtSvc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("não deveria abrir o stream")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("got %d, want 403", rec.Code)
	}
}

// O login precisa anunciar a marca, senão a tela não sabe que deve obrigar.
func TestLogin_AnunciaSenhaProvisoria(t *testing.T) {
	repo := novoRepoSenha("senha-antiga-1", true)
	repo.grupos = []GrupoInfo{{ID: gA}}

	resp, err := newTestService(repo).Login(context.Background(), "t@e.com", "senha-antiga-1")
	if err != nil {
		t.Fatalf("%v", err)
	}
	if !resp.SenhaProvisoria {
		t.Fatal("a resposta do login não anuncia a senha provisória")
	}
	if c := claimsDoToken(t, resp.AccessToken); !c.SenhaProvisoria {
		t.Fatal("o token não carrega a marca, então a trava não vale")
	}
}
