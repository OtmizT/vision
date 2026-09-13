package grupos

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

// mockService para handler tests
type mockService struct {
	grupo     *Grupo
	grupos    []*Grupo
	total     int64
	createErr error
	updateErr error
	deleteErr error
	getErr    error
}

func (m *mockService) Create(_ context.Context, _ CreateRequest) (*Grupo, error) {
	return m.grupo, m.createErr
}
func (m *mockService) GetByID(_ context.Context, _ string) (*Grupo, error) {
	return m.grupo, m.getErr
}
func (m *mockService) List(_ context.Context, _ ListParams) ([]*Grupo, int64, error) {
	return m.grupos, m.total, nil
}
func (m *mockService) Update(_ context.Context, _ string, _ UpdateRequest) (*Grupo, error) {
	return m.grupo, m.updateErr
}
func (m *mockService) Delete(_ context.Context, _ string) error {
	return m.deleteErr
}

func newTestHandler(svc Service) *Handler {
	return NewHandler(svc, auth.NewJWTService(testJWTSecret), nil)
}

func bearerToken(t *testing.T, role string) string {
	t.Helper()
	jwtSvc := auth.NewJWTService(testJWTSecret)
	token, err := jwtSvc.Generate("user-1", "grupo-1", "u@test.com", role)
	if err != nil {
		t.Fatalf("gerar token: %v", err)
	}
	return "Bearer " + token
}

func doReq(t *testing.T, handler http.Handler, method, path string, body any, authHeader string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func TestHandler_List_RequiresAuth(t *testing.T) {
	h := newTestHandler(&mockService{})
	rr := doReq(t, h.Routes(), http.MethodGet, "/", nil, "")
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d want 401", rr.Code)
	}
}

func TestHandler_List_Success(t *testing.T) {
	svc := &mockService{
		grupos: []*Grupo{{ID: "g1", Nome: "Alpha"}, {ID: "g2", Nome: "Beta"}},
		total:  2,
	}
	h := newTestHandler(svc)

	rr := doReq(t, h.Routes(), http.MethodGet, "/", nil, bearerToken(t, "admin_global"))
	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200 — %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["success"] != true {
		t.Error("success deveria ser true")
	}
	meta := resp["meta"].(map[string]any)
	if meta["total"].(float64) != 2 {
		t.Errorf("meta.total: got %v want 2", meta["total"])
	}
}

func TestHandler_Create_Success(t *testing.T) {
	svc := &mockService{grupo: &Grupo{ID: "new", Nome: "Alpha", Slug: "alpha"}}
	h := newTestHandler(svc)

	rr := doReq(t, h.Routes(), http.MethodPost, "/",
		map[string]string{"nome": "Alpha", "slug": "alpha"},
		bearerToken(t, "admin_global"))

	if rr.Code != http.StatusCreated {
		t.Fatalf("status: got %d want 201 — %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_Create_Conflict(t *testing.T) {
	svc := &mockService{createErr: apperror.Conflict("slug duplicado")}
	h := newTestHandler(svc)

	rr := doReq(t, h.Routes(), http.MethodPost, "/",
		map[string]string{"nome": "Alpha", "slug": "alpha"},
		bearerToken(t, "admin_global"))

	if rr.Code != http.StatusConflict {
		t.Fatalf("status: got %d want 409", rr.Code)
	}
}

func TestHandler_GetByID_NotFound(t *testing.T) {
	svc := &mockService{getErr: apperror.NotFound("grupo não encontrado")}
	h := newTestHandler(svc)

	rr := doReq(t, h.Routes(), http.MethodGet, "/some-id", nil, bearerToken(t, "admin_grupo"))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status: got %d want 404", rr.Code)
	}
}

func TestHandler_GetByID_Success(t *testing.T) {
	svc := &mockService{grupo: &Grupo{ID: "g1", Nome: "Alpha"}}
	h := newTestHandler(svc)

	rr := doReq(t, h.Routes(), http.MethodGet, "/g1", nil, bearerToken(t, "admin_grupo"))
	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200", rr.Code)
	}
}

func TestHandler_Update_Success(t *testing.T) {
	svc := &mockService{grupo: &Grupo{ID: "g1", Nome: "Novo Nome"}}
	h := newTestHandler(svc)

	rr := doReq(t, h.Routes(), http.MethodPut, "/g1",
		map[string]string{"nome": "Novo Nome"},
		bearerToken(t, "admin_grupo"))

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200", rr.Code)
	}
}

func TestHandler_Delete_Success(t *testing.T) {
	svc := &mockService{}
	h := newTestHandler(svc)

	rr := doReq(t, h.Routes(), http.MethodDelete, "/g1", nil, bearerToken(t, "admin_grupo"))
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status: got %d want 204", rr.Code)
	}
}

func TestHandler_Delete_WithActiveEmpresas(t *testing.T) {
	svc := &mockService{deleteErr: apperror.Conflict("grupo possui empresas ativas")}
	h := newTestHandler(svc)

	rr := doReq(t, h.Routes(), http.MethodDelete, "/g1", nil, bearerToken(t, "admin_grupo"))
	if rr.Code != http.StatusConflict {
		t.Fatalf("status: got %d want 409", rr.Code)
	}
}

func TestHandler_ViewerCannotDelete(t *testing.T) {
	svc := &mockService{}
	h := newTestHandler(svc)

	rr := doReq(t, h.Routes(), http.MethodDelete, "/g1", nil, bearerToken(t, "viewer"))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status: got %d want 403", rr.Code)
	}
}
