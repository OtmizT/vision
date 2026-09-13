package audit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestAtor_EscritaAtravessaContextos(t *testing.T) {
	ctx, ator := ComAtor(context.Background())

	// Simula o que o RequireAuth faz: deriva outro contexto e escreve nele. A
	// escrita precisa aparecer no ponteiro original, senão a auditoria continua
	// gravando request sem dono.
	filho := context.WithValue(ctx, contextKey("qualquer"), 1)
	AtorFromContext(filho).Registrar("u-1", "a@b.c", "admin_grupo")

	if ator.UserID != "u-1" || ator.Email != "a@b.c" || ator.Role != "admin_grupo" {
		t.Fatalf("ator não propagou: %+v", *ator)
	}
}

func TestAtor_NilNaoQuebra(t *testing.T) {
	// Autenticação também roda onde não há middleware de auditoria.
	AtorFromContext(context.Background()).Registrar("u-1", "a@b.c", "viewer")
}

// repoFake guarda a última entrada gravada.
type repoFake struct {
	ch chan LogEntry
}

func (r *repoFake) Insert(ctx context.Context, e LogEntry) error {
	r.ch <- e
	return nil
}

func TestMiddleware_GravaQuemFez(t *testing.T) {
	repo := &repoFake{ch: make(chan LogEntry, 1)}
	h := Middleware(repo, zerolog.Nop())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// O papel do RequireAuth.
		AtorFromContext(r.Context()).Registrar("u-7", "ana@alpha.com", "admin_grupo")
		w.WriteHeader(http.StatusOK)
	}))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/admin/grupos", nil))

	e := <-repo.ch
	if e.UserID != "u-7" || e.UserEmail != "ana@alpha.com" || e.Role != "admin_grupo" {
		t.Fatalf("trilha sem autor: %+v", e)
	}
}

// Rota pública continua sendo auditada — só que sem autor.
func TestMiddleware_RotaSemAuthGravaSemAutor(t *testing.T) {
	repo := &repoFake{ch: make(chan LogEntry, 1)}
	h := Middleware(repo, zerolog.Nop())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/health", nil))

	e := <-repo.ch
	if e.UserID != "" || e.Path != "/health" {
		t.Fatalf("got %+v", e)
	}
}

// O motivo de sanitizar a query: as rotas SSE aceitam ?token=<JWT>.
func TestMiddleware_NaoGravaTokenDaQueryString(t *testing.T) {
	repo := &repoFake{ch: make(chan LogEntry, 1)}
	h := Middleware(repo, zerolog.Nop())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	h.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest(http.MethodGet, "/sync/1/stream?token=eyJhbGciOiJIUzI1NiJ9.abc.def&aba=progresso", nil))

	e := <-repo.ch
	if strings.Contains(e.QueryParams, "eyJhbGciOiJIUzI1NiJ9") {
		t.Fatalf("token gravado em claro: %q", e.QueryParams)
	}
	if !strings.Contains(e.QueryParams, "aba=progresso") {
		t.Fatalf("perdeu o resto da query: %q", e.QueryParams)
	}
}
