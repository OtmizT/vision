package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"omie-sync-api/internal/audit"
)

// checkerFake devolve o vínculo combinado, ou um erro, sem tocar em banco.
type checkerFake struct {
	pertence bool
	err      error
	chamadas int
}

func (c *checkerFake) ValidateUsuarioGrupo(ctx context.Context, usuarioID, grupoID string) (bool, error) {
	c.chamadas++
	return c.pertence, c.err
}

// rodaGrupoMembro monta /g/{grupoID} com as claims já no contexto e devolve o
// status e se o handler final chegou a rodar.
func rodaGrupoMembro(t *testing.T, membros MembroChecker, claims *JWTClaims, grupoURL string) (int, bool) {
	t.Helper()

	chegou := false
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if claims == nil {
				next.ServeHTTP(w, req)
				return
			}
			next.ServeHTTP(w, req.WithContext(context.WithValue(req.Context(), CtxKeyUserClaims, claims)))
		})
	})
	r.With(RequireGrupoMembro(membros)).Get("/g/{grupoID}", func(w http.ResponseWriter, req *http.Request) {
		chegou = true
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/g/"+grupoURL, nil))
	return rec.Code, chegou
}

const (
	grupoA = "11111111-1111-1111-1111-111111111111"
	grupoB = "22222222-2222-2222-2222-222222222222"
)

func claimsDe(role, grupo string) *JWTClaims {
	return &JWTClaims{UserID: "u-1", Email: "a@b.c", Role: role, GrupoID: grupo}
}

func TestRequireGrupoMembro(t *testing.T) {
	t.Run("membro do próprio grupo passa", func(t *testing.T) {
		c := &checkerFake{pertence: true}
		code, chegou := rodaGrupoMembro(t, c, claimsDe("admin_grupo", grupoA), grupoA)
		if code != http.StatusOK || !chegou {
			t.Fatalf("got %d chegou=%v, want 200 true", code, chegou)
		}
		if c.chamadas != 1 {
			t.Fatalf("o vínculo precisa ser consultado: chamadas=%d", c.chamadas)
		}
	})

	// O vazamento entre clientes: trocar o UUID da URL.
	t.Run("grupo de outro cliente na URL é negado", func(t *testing.T) {
		c := &checkerFake{pertence: true}
		code, chegou := rodaGrupoMembro(t, c, claimsDe("admin_grupo", grupoA), grupoB)
		if code != http.StatusForbidden || chegou {
			t.Fatalf("got %d chegou=%v, want 403 false", code, chegou)
		}
	})

	// Tirar alguém do grupo passa a valer agora, e não quando o token expirar.
	t.Run("token válido sem vínculo no banco é negado", func(t *testing.T) {
		c := &checkerFake{pertence: false}
		code, chegou := rodaGrupoMembro(t, c, claimsDe("admin_grupo", grupoA), grupoA)
		if code != http.StatusForbidden || chegou {
			t.Fatalf("got %d chegou=%v, want 403 false", code, chegou)
		}
	})

	// Liberar em caso de erro transformaria banco fora do ar em acesso livre.
	t.Run("erro ao consultar fecha a porta", func(t *testing.T) {
		c := &checkerFake{err: errors.New("conexão recusada")}
		code, chegou := rodaGrupoMembro(t, c, claimsDe("admin_grupo", grupoA), grupoA)
		if code != http.StatusInternalServerError || chegou {
			t.Fatalf("got %d chegou=%v, want 500 false", code, chegou)
		}
	})

	t.Run("admin_global passa sem consultar vínculo", func(t *testing.T) {
		c := &checkerFake{pertence: false}
		code, chegou := rodaGrupoMembro(t, c, claimsDe("admin_global", ""), grupoB)
		if code != http.StatusOK || !chegou {
			t.Fatalf("got %d chegou=%v, want 200 true", code, chegou)
		}
		if c.chamadas != 0 {
			t.Fatalf("admin_global não deveria consultar: chamadas=%d", c.chamadas)
		}
	})

	t.Run("sem claims é 401", func(t *testing.T) {
		code, chegou := rodaGrupoMembro(t, &checkerFake{pertence: true}, nil, grupoA)
		if code != http.StatusUnauthorized || chegou {
			t.Fatalf("got %d chegou=%v, want 401 false", code, chegou)
		}
	})

	// Montagem sem banco mantém o comportamento antigo — nunca mais frouxa.
	t.Run("checker nil ainda compara o grupo do token", func(t *testing.T) {
		if code, _ := rodaGrupoMembro(t, nil, claimsDe("admin_grupo", grupoA), grupoA); code != http.StatusOK {
			t.Fatalf("mesmo grupo: got %d, want 200", code)
		}
		if code, _ := rodaGrupoMembro(t, nil, claimsDe("admin_grupo", grupoA), grupoB); code != http.StatusForbidden {
			t.Fatalf("outro grupo: got %d, want 403", code)
		}
	})
}

// O middleware de auditoria roda antes da autenticação e não tem como saber
// quem é; o RequireAuth preenche o Ator que ele deixou no contexto.
func TestRequireAuth_RegistraAtorParaAuditoria(t *testing.T) {
	jwtSvc := NewJWTService("segredo-de-teste-com-mais-de-32-caracteres")
	token, err := jwtSvc.Generate("u-9", grupoA, "ana@alpha.com", "admin_grupo", ContextoGrupo)
	if err != nil {
		t.Fatalf("gerar token: %v", err)
	}

	ctx, ator := audit.ComAtor(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/x", nil).WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+token)

	h := RequireAuth(jwtSvc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	h.ServeHTTP(httptest.NewRecorder(), req)

	if ator.UserID != "u-9" || ator.Email != "ana@alpha.com" || ator.Role != "admin_grupo" {
		t.Fatalf("ator não preenchido: %+v", *ator)
	}
}

// Token inválido não pode carimbar ninguém na trilha de auditoria.
func TestRequireAuth_TokenInvalidoNaoRegistraAtor(t *testing.T) {
	jwtSvc := NewJWTService("segredo-de-teste-com-mais-de-32-caracteres")
	ctx, ator := audit.ComAtor(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/x", nil).WithContext(ctx)
	req.Header.Set("Authorization", "Bearer lixo")

	h := RequireAuth(jwtSvc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("não deveria chegar ao handler")
	}))
	h.ServeHTTP(httptest.NewRecorder(), req)

	if ator.UserID != "" {
		t.Fatalf("ator preenchido com token inválido: %+v", *ator)
	}
}

// Sem middleware de auditoria no contexto não há Ator, e autenticar não pode
// entrar em pânico por causa disso.
func TestRequireAuth_SemAtorNoContexto(t *testing.T) {
	jwtSvc := NewJWTService("segredo-de-teste-com-mais-de-32-caracteres")
	token, _ := jwtSvc.Generate("u-1", grupoA, "v@x.c", "viewer", ContextoGrupo)
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	ok := false
	h := RequireAuth(jwtSvc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { ok = true }))
	h.ServeHTTP(httptest.NewRecorder(), req)

	if !ok {
		t.Fatal("handler não rodou")
	}
}
