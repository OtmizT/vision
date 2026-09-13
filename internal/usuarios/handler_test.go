package usuarios

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"omie-sync-api/internal/apperror"
	"omie-sync-api/internal/auth"
)

const testJWTSecret = "test-secret-minimo-32-caracteres-xpto"

type mockSvc struct {
	usuario  *Usuario
	usuarios []*Usuario
	total    int64
	err      error
}

func (m *mockSvc) Create(_ context.Context, _ string, _ CreateRequest) (*CreateResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &CreateResult{Usuario: m.usuario, AddedToGroup: false}, nil
}
func (m *mockSvc) GetByID(_ context.Context, _ string) (*Usuario, error) {
	return m.usuario, m.err
}
func (m *mockSvc) List(_ context.Context, _ ListParams) ([]*Usuario, int64, error) {
	return m.usuarios, m.total, m.err
}
func (m *mockSvc) Update(_ context.Context, _, _ string, _ UpdateRequest) (*Usuario, error) {
	return m.usuario, m.err
}
func (m *mockSvc) UpdatePassword(_ context.Context, _ string, _ UpdatePasswordRequest) error {
	return m.err
}
func (m *mockSvc) Delete(_ context.Context, _ string) error { return m.err }

func newTestHandler(svc Service) *Handler {
	return NewHandler(svc, auth.NewJWTService(testJWTSecret), nil)
}

func token(t *testing.T, role string) string {
	t.Helper()
	tok, _ := auth.NewJWTService(testJWTSecret).Generate("u1", "g1", "u@t.com", role)
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

func sampleUser() *Usuario {
	return &Usuario{ID: "u1", GrupoID: "g1", Nome: "Ana", Email: "ana@t.com", Role: "viewer", Ativo: true}
}

func TestHandler_RequiresAuth(t *testing.T) {
	rr := doReq(t, newTestHandler(&mockSvc{}).Routes(), http.MethodGet, "/", nil, "")
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("got %d want 401", rr.Code)
	}
}

func TestHandler_ViewerForbidden(t *testing.T) {
	rr := doReq(t, newTestHandler(&mockSvc{}).Routes(), http.MethodGet, "/", nil, token(t, "viewer"))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("got %d want 403", rr.Code)
	}
}

func TestHandler_List_Success(t *testing.T) {
	svc := &mockSvc{usuarios: []*Usuario{sampleUser()}, total: 1}
	rr := doReq(t, newTestHandler(svc).Routes(), http.MethodGet, "/", nil, token(t, "admin_grupo"))
	if rr.Code != http.StatusOK {
		t.Fatalf("got %d want 200 — %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_Create_Success(t *testing.T) {
	svc := &mockSvc{usuario: sampleUser()}
	rr := doReq(t, newTestHandler(svc).Routes(), http.MethodPost, "/",
		CreateRequest{Nome: "Ana", Email: "ana@t.com", Password: "senha123"},
		token(t, "admin_grupo"))
	if rr.Code != http.StatusCreated {
		t.Fatalf("got %d want 201 — %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_Create_PasswordNotInResponse(t *testing.T) {
	svc := &mockSvc{usuario: sampleUser()}
	rr := doReq(t, newTestHandler(svc).Routes(), http.MethodPost, "/",
		CreateRequest{Nome: "X", Email: "x@t.com", Password: "segredo123"},
		token(t, "admin_grupo"))
	if contains(rr.Body.String(), "segredo123") {
		t.Error("password não deve aparecer na response")
	}
}

func TestHandler_GetByID_NotFound(t *testing.T) {
	svc := &mockSvc{err: apperror.NotFound("não encontrado")}
	rr := doReq(t, newTestHandler(svc).Routes(), http.MethodGet, "/u1", nil, token(t, "admin_grupo"))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("got %d want 404", rr.Code)
	}
}

func TestHandler_UpdatePassword_NoContent(t *testing.T) {
	rr := doReq(t, newTestHandler(&mockSvc{}).Routes(), http.MethodPut, "/u1/password",
		UpdatePasswordRequest{Password: "novasenha123"},
		token(t, "admin_grupo"))
	if rr.Code != http.StatusNoContent {
		t.Fatalf("got %d want 204 — %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_Delete_Success(t *testing.T) {
	rr := doReq(t, newTestHandler(&mockSvc{}).Routes(), http.MethodDelete, "/u1", nil, token(t, "admin_grupo"))
	if rr.Code != http.StatusNoContent {
		t.Fatalf("got %d want 204", rr.Code)
	}
}

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
