package empresas

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
	empresa   *EmpresaResponse
	empresas  []*EmpresaResponse
	total     int64
	createErr error
	updateErr error
	deleteErr error
	getErr    error
	reativarErr error
}

func (m *mockSvc) Create(_ context.Context, _ string, _ CreateRequest) (*EmpresaResponse, error) {
	return m.empresa, m.createErr
}
func (m *mockSvc) GetByID(_ context.Context, _ string) (*EmpresaResponse, error) {
	return m.empresa, m.getErr
}
func (m *mockSvc) List(_ context.Context, _ ListParams) ([]*EmpresaResponse, int64, error) {
	return m.empresas, m.total, nil
}
func (m *mockSvc) Update(_ context.Context, _ string, _ UpdateRequest) (*EmpresaResponse, error) {
	return m.empresa, m.updateErr
}
func (m *mockSvc) Delete(_ context.Context, _ string) error   { return m.deleteErr }
func (m *mockSvc) Reativar(_ context.Context, _ string) error { return m.reativarErr }

func newTestHandler(svc Service) *Handler {
	return NewHandler(svc, auth.NewJWTService(testJWTSecret), nil)
}

func bearerToken(t *testing.T, role string) string {
	t.Helper()
	jwtSvc := auth.NewJWTService(testJWTSecret)
	tok, _ := jwtSvc.Generate("u1", "g1", "u@t.com", role, auth.ContextoGrupo)
	return "Bearer " + tok
}

func doReq(t *testing.T, h http.Handler, method, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func sampleEmpresaResponse() *EmpresaResponse {
	return &EmpresaResponse{
		ID: "emp-1", GrupoID: "grp-1", Nome: "Acme",
		AppKey: "key1", AppSecret: "key1****",
		Status: "ativa", StatusSync: "ativo",
	}
}

func TestHandler_RequiresAuth(t *testing.T) {
	h := newTestHandler(&mockSvc{})
	rr := doReq(t, h.Routes(), http.MethodGet, "/", nil, "")
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d want 401", rr.Code)
	}
}

func TestHandler_ViewerForbidden(t *testing.T) {
	h := newTestHandler(&mockSvc{})
	rr := doReq(t, h.Routes(), http.MethodGet, "/", nil, bearerToken(t, "viewer"))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status: got %d want 403", rr.Code)
	}
}

func TestHandler_List_Success(t *testing.T) {
	svc := &mockSvc{empresas: []*EmpresaResponse{sampleEmpresaResponse()}, total: 1}
	rr := doReq(t, newTestHandler(svc).Routes(), http.MethodGet, "/", nil, bearerToken(t, "admin_grupo"))
	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200 — %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_Create_Success(t *testing.T) {
	svc := &mockSvc{empresa: sampleEmpresaResponse()}
	rr := doReq(t, newTestHandler(svc).Routes(), http.MethodPost, "/",
		map[string]string{"nome": "Acme", "app_key": "k", "app_secret": "s"},
		bearerToken(t, "admin_grupo"))
	if rr.Code != http.StatusCreated {
		t.Fatalf("status: got %d want 201 — %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_Create_AppSecretNotInResponse(t *testing.T) {
	svc := &mockSvc{empresa: sampleEmpresaResponse()}
	rr := doReq(t, newTestHandler(svc).Routes(), http.MethodPost, "/",
		map[string]string{"nome": "Acme", "app_key": "k", "app_secret": "supersecret"},
		bearerToken(t, "admin_grupo"))

	body := rr.Body.String()
	if contains(body, "supersecret") {
		t.Error("app_secret exposto na response")
	}
}

func TestHandler_GetByID_NotFound(t *testing.T) {
	svc := &mockSvc{getErr: apperror.NotFound("não encontrado")}
	rr := doReq(t, newTestHandler(svc).Routes(), http.MethodGet, "/emp-1", nil, bearerToken(t, "admin_grupo"))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status: got %d want 404", rr.Code)
	}
}

func TestHandler_Delete_Success(t *testing.T) {
	rr := doReq(t, newTestHandler(&mockSvc{}).Routes(), http.MethodDelete, "/emp-1", nil, bearerToken(t, "admin_grupo"))
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status: got %d want 204 — %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_Delete_AlreadyDeletando(t *testing.T) {
	svc := &mockSvc{deleteErr: apperror.Conflict("já em exclusão")}
	rr := doReq(t, newTestHandler(svc).Routes(), http.MethodDelete, "/emp-1", nil, bearerToken(t, "admin_grupo"))
	if rr.Code != http.StatusConflict {
		t.Fatalf("status: got %d want 409", rr.Code)
	}
}

func TestHandler_Reativar_Success(t *testing.T) {
	rr := doReq(t, newTestHandler(&mockSvc{}).Routes(), http.MethodPost, "/emp-1/reativar", nil, bearerToken(t, "admin_grupo"))
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status: got %d want 204 — %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_Reativar_JaAtiva(t *testing.T) {
	svc := &mockSvc{reativarErr: apperror.Conflict("já ativa")}
	rr := doReq(t, newTestHandler(svc).Routes(), http.MethodPost, "/emp-1/reativar", nil, bearerToken(t, "admin_grupo"))
	if rr.Code != http.StatusConflict {
		t.Fatalf("status: got %d want 409", rr.Code)
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
