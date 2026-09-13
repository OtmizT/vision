package permissoes

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"omie-sync-api/internal/apperror"
	"omie-sync-api/internal/auth"
)

const testJWTSecret = "test-secret-minimo-32-caracteres-xpto"

type mockSvc struct {
	permissao  *Permissao
	permissoes []*Permissao
	grantErr   error
	revokeErr  error
	listErr    error
}

func (m *mockSvc) Grant(_ context.Context, _ GrantRequest) (*Permissao, error) {
	return m.permissao, m.grantErr
}
func (m *mockSvc) Revoke(_ context.Context, _ RevokeRequest) error { return m.revokeErr }
func (m *mockSvc) ListByUsuario(_ context.Context, _ string) ([]*Permissao, error) {
	return m.permissoes, m.listErr
}
func (m *mockSvc) ListByEmpresa(_ context.Context, _ string) ([]*Permissao, error) {
	return m.permissoes, m.listErr
}

func newTestHandler(svc Service) *Handler {
	return NewHandler(svc, auth.NewJWTService(testJWTSecret))
}

func token(t *testing.T, role string) string {
	t.Helper()
	tok, _ := auth.NewJWTService(testJWTSecret).Generate("u1", "g1", "u@t.com", role, auth.ContextoGrupo, false)
	return "Bearer " + tok
}

func doReq(t *testing.T, h http.Handler, method, path string, body any, tok string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if tok != "" {
		req.Header.Set("Authorization", tok)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func samplePerm() *Permissao {
	return &Permissao{ID: "p1", UsuarioID: "u1", EmpresaID: "e1", Recurso: "sync", Acao: "ver", CreatedAt: time.Now()}
}

func TestHandler_Grant_RequiresAuth(t *testing.T) {
	rr := doReq(t, newTestHandler(&mockSvc{}).Routes(), http.MethodPost, "/grant", nil, "")
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("got %d want 401", rr.Code)
	}
}

func TestHandler_Grant_ViewerForbidden(t *testing.T) {
	rr := doReq(t, newTestHandler(&mockSvc{}).Routes(), http.MethodPost, "/grant", nil, token(t, "viewer"))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("got %d want 403", rr.Code)
	}
}

func TestHandler_Grant_Success(t *testing.T) {
	svc := &mockSvc{permissao: samplePerm()}
	rr := doReq(t, newTestHandler(svc).Routes(), http.MethodPost, "/grant",
		GrantRequest{UsuarioID: "u1", EmpresaID: "e1", Recurso: "sync", Acao: "ver"},
		token(t, "admin_grupo"))
	if rr.Code != http.StatusCreated {
		t.Fatalf("got %d want 201 — %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_Grant_InvalidBody(t *testing.T) {
	svc := &mockSvc{grantErr: apperror.Unprocessable("recurso inválido")}
	rr := doReq(t, newTestHandler(svc).Routes(), http.MethodPost, "/grant",
		GrantRequest{UsuarioID: "u1", EmpresaID: "e1", Recurso: "invalido", Acao: "ver"},
		token(t, "admin_grupo"))
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("got %d want 422 — %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_Revoke_Success(t *testing.T) {
	rr := doReq(t, newTestHandler(&mockSvc{}).Routes(), http.MethodPost, "/revoke",
		RevokeRequest{UsuarioID: "u1", EmpresaID: "e1", Recurso: "sync", Acao: "ver"},
		token(t, "admin_grupo"))
	if rr.Code != http.StatusNoContent {
		t.Fatalf("got %d want 204 — %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_ListByUsuario_Success(t *testing.T) {
	svc := &mockSvc{permissoes: []*Permissao{samplePerm()}}
	rr := doReq(t, newTestHandler(svc).Routes(), http.MethodGet, "/usuario/u1", nil, token(t, "admin_grupo"))
	if rr.Code != http.StatusOK {
		t.Fatalf("got %d want 200 — %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_ListByEmpresa_Success(t *testing.T) {
	svc := &mockSvc{permissoes: []*Permissao{samplePerm()}}
	rr := doReq(t, newTestHandler(svc).Routes(), http.MethodGet, "/empresa/e1", nil, token(t, "admin_grupo"))
	if rr.Code != http.StatusOK {
		t.Fatalf("got %d want 200 — %s", rr.Code, rr.Body.String())
	}
}
