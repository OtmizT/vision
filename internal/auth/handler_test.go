package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// mockService implementa Service para testes do handler
type mockService struct {
	loginResp   *LoginResponse
	loginErr    error
	logoutErr   error
	refreshResp *LoginResponse
	refreshErr  error
	meResp      *MeResponse
	meErr       error
}

func (m *mockService) Login(_ context.Context, _, _ string) (*LoginResponse, error) {
	return m.loginResp, m.loginErr
}
func (m *mockService) SelectGrupo(_ context.Context, _ string, _ Contexto, _ string) (*LoginResponse, error) {
	return m.loginResp, m.loginErr
}
func (m *mockService) TrocaGrupo(_ context.Context, _ string, _ Contexto, _ string) (*LoginResponse, error) {
	return m.loginResp, m.loginErr
}
func (m *mockService) GetGrupos(_ context.Context, _ string) ([]GrupoInfo, error) {
	return nil, nil
}
func (m *mockService) TrocarSenhaPropria(_ context.Context, _ string, _ Contexto, _ string, _ TrocaSenhaRequest) (*LoginResponse, error) {
	return m.loginResp, m.loginErr
}
func (m *mockService) GetContextos(_ context.Context, _ string) (*ContextosResponse, error) {
	return &ContextosResponse{}, nil
}
func (m *mockService) Logout(_ context.Context, _ string) error { return m.logoutErr }
func (m *mockService) Refresh(_ context.Context, _ string) (*LoginResponse, error) {
	return m.refreshResp, m.refreshErr
}
func (m *mockService) Me(_ context.Context, _ string) (*MeResponse, error) {
	return m.meResp, m.meErr
}

func newTestHandler(svc Service) *Handler {
	return NewHandler(svc, NewJWTService(testSecret))
}

func doRequest(t *testing.T, handler http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func TestHandler_Login_Success(t *testing.T) {
	svc := &mockService{loginResp: &LoginResponse{
		AccessToken: "acc", RefreshToken: "ref", ExpiresIn: 900,
	}}
	h := newTestHandler(svc)

	rr := doRequest(t, h.Routes(), http.MethodPost, "/login", map[string]string{
		"email": "u@test.com", "password": "pass",
	})

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200", rr.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["success"] != true {
		t.Error("success deveria ser true")
	}
}

func TestHandler_Login_MissingFields(t *testing.T) {
	h := newTestHandler(&mockService{})

	rr := doRequest(t, h.Routes(), http.MethodPost, "/login", map[string]string{"email": ""})
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status: got %d want 422", rr.Code)
	}
}

func TestHandler_Logout_Success(t *testing.T) {
	h := newTestHandler(&mockService{})

	rr := doRequest(t, h.Routes(), http.MethodPost, "/logout", map[string]string{
		"refresh_token": "some-token",
	})
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status: got %d want 204", rr.Code)
	}
}

func TestHandler_Refresh_Success(t *testing.T) {
	svc := &mockService{refreshResp: &LoginResponse{
		AccessToken: "new-acc", RefreshToken: "new-ref", ExpiresIn: 900,
	}}
	h := newTestHandler(svc)

	rr := doRequest(t, h.Routes(), http.MethodPost, "/refresh", map[string]string{
		"refresh_token": "old-token",
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200", rr.Code)
	}
}

func TestHandler_Me_RequiresAuth(t *testing.T) {
	h := newTestHandler(&mockService{})

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	rr := httptest.NewRecorder()
	h.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d want 401", rr.Code)
	}
}

func TestHandler_Me_WithValidToken(t *testing.T) {
	svc := &mockService{meResp: &MeResponse{
		ID: "u1", GrupoID: "g1", Nome: "Test", Email: "t@t.com", Role: "viewer",
	}}
	h := newTestHandler(svc)

	// Gera token válido
	jwtSvc := NewJWTService(testSecret)
	token, _ := jwtSvc.Generate("u1", "g1", "t@t.com", "viewer", ContextoGrupo, false)

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	h.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200 — body: %s", rr.Code, rr.Body.String())
	}
}

var _ = time.Now // evitar import não usado
