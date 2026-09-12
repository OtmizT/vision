package sync

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
	"omie-sync-api/internal/webhooks"
)

const testJWTSecret = "test-secret-minimo-32-caracteres-xpto"

// mock service para handler

type mockSvc struct {
	status        *StatusResponse
	jobs          []*SyncJob
	total         int64
	job           *SyncJob
	control       *SyncControl
	progress      []*SyncJobProgress
	statusErr     error
	listErr       error
	forcarErr     error
	configurarErr error
	progressErr   error
}

func (m *mockSvc) GetStatus(_ context.Context, empresaID string) (*StatusResponse, error) {
	return m.status, m.statusErr
}
func (m *mockSvc) ListJobs(_ context.Context, _ ListParams) ([]*SyncJob, int64, error) {
	return m.jobs, m.total, m.listErr
}
func (m *mockSvc) ForcarSync(_ context.Context, _, _ string, _ ForcarSyncRequest) (*SyncJob, error) {
	return m.job, m.forcarErr
}
func (m *mockSvc) Configurar(_ context.Context, _ string, _ ConfigurarRequest) (*SyncControl, error) {
	return m.control, m.configurarErr
}
func (m *mockSvc) GetJobProgress(_ context.Context, _ string) ([]*SyncJobProgress, error) {
	return m.progress, m.progressErr
}
func (m *mockSvc) UltimaAtualizacao(_ context.Context, _ string) (*UltimaAtualizacao, error) {
	return &UltimaAtualizacao{}, nil
}
func (m *mockSvc) GetExecutorConfigs(_ context.Context, _ string) ([]*EmpresaExecutorConfig, error) {
	return nil, nil
}
func (m *mockSvc) UpdateExecutorConfig(_ context.Context, _, _ string, _ UpdateExecutorConfigRequest, _ string) (*EmpresaExecutorConfig, error) {
	return nil, nil
}
func (m *mockSvc) StartupRecovery(_ context.Context) error                      { return nil }
func (m *mockSvc) GetAdminOverview(_ context.Context) (map[string]int64, error) { return nil, nil }
func (m *mockSvc) GetJobsAtivos(_ context.Context) ([]JobAtivoRow, error)       { return nil, nil }
func (m *mockSvc) ListEmpresasSyncGeral(_ context.Context) ([]EmpresaSyncRow, error) {
	return nil, nil
}
func (m *mockSvc) CancelarJob(_ context.Context, _ string) error       { return nil }
func (m *mockSvc) GetDLQPages(_ context.Context) ([]DLQPageRow, error) { return nil, nil }
func (m *mockSvc) RetryDLQPage(_ context.Context, _ string) error      { return nil }
func (m *mockSvc) GetPagesByEmpresa(_ context.Context, _, _ string) ([]PageRow, error) {
	return nil, nil
}

func newTestHandler(svc Service) *Handler {
	return NewHandler(svc, auth.NewJWTService(testJWTSecret), NewSSEHub())
}

func bearerToken(t *testing.T, role string) string {
	t.Helper()
	tok, _ := auth.NewJWTService(testJWTSecret).Generate("u1", "g1", "u@t.com", role)
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

func TestHandler_GetStatus_RequiresAuth(t *testing.T) {
	h := newTestHandler(&mockSvc{})
	rr := doReq(t, h.Routes(), http.MethodGet, "/emp-1/status", nil, "")
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d want 401", rr.Code)
	}
}

func TestHandler_GetStatus_Success(t *testing.T) {
	svc := &mockSvc{status: &StatusResponse{
		EmpresaID: "emp-1",
		Control:   &SyncControl{Ativo: true, IntervaloIncrementalMin: 60, IntervaloFullDias: 7},
	}}
	rr := doReq(t, newTestHandler(svc).Routes(), http.MethodGet, "/emp-1/status", nil, bearerToken(t, "viewer"))
	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200 — %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_ListJobs_Success(t *testing.T) {
	svc := &mockSvc{
		jobs:  []*SyncJob{{ID: "j1", Status: "concluido"}},
		total: 1,
	}
	rr := doReq(t, newTestHandler(svc).Routes(), http.MethodGet, "/emp-1/jobs", nil, bearerToken(t, "viewer"))
	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200 — %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	meta := resp["meta"].(map[string]any)
	if meta["total"].(float64) != 1 {
		t.Errorf("meta.total: got %v want 1", meta["total"])
	}
}

func TestHandler_ForcarSync_RequiresAdminRole(t *testing.T) {
	h := newTestHandler(&mockSvc{})
	rr := doReq(t, h.Routes(), http.MethodPost, "/emp-1/forcar", nil, bearerToken(t, "viewer"))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status: got %d want 403", rr.Code)
	}
}

func TestHandler_ForcarSync_Success(t *testing.T) {
	now := time.Now()
	svc := &mockSvc{job: &SyncJob{ID: "j1", Tipo: "manual", Status: "pendente", CreatedAt: now}}
	rr := doReq(t, newTestHandler(svc).Routes(), http.MethodPost, "/emp-1/forcar",
		map[string]string{"tipo": "manual"}, bearerToken(t, "admin_grupo"))
	if rr.Code != http.StatusCreated {
		t.Fatalf("status: got %d want 201 — %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_Configurar_RequiresAdminRole(t *testing.T) {
	h := newTestHandler(&mockSvc{})
	rr := doReq(t, h.Routes(), http.MethodPut, "/emp-1/configurar", nil, bearerToken(t, "viewer"))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status: got %d want 403", rr.Code)
	}
}

func TestHandler_Configurar_Success(t *testing.T) {
	svc := &mockSvc{control: &SyncControl{Ativo: true, IntervaloIncrementalMin: 60, IntervaloFullDias: 7}}
	rr := doReq(t, newTestHandler(svc).Routes(), http.MethodPut, "/emp-1/configurar",
		ConfigurarRequest{Ativo: true, IntervaloIncrementalMin: 60, IntervaloFullDias: 7}, bearerToken(t, "admin_grupo"))
	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200 — %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_Configurar_IntervaloInvalido(t *testing.T) {
	svc := &mockSvc{configurarErr: apperror.Unprocessable("intervalo inválido")}
	rr := doReq(t, newTestHandler(svc).Routes(), http.MethodPut, "/emp-1/configurar",
		ConfigurarRequest{Ativo: true, IntervaloIncrementalMin: 30, IntervaloFullDias: 1}, bearerToken(t, "admin_grupo"))
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status: got %d want 422 — %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_GetJobProgress_Success(t *testing.T) {
	svc := &mockSvc{progress: []*SyncJobProgress{{Executor: "clientes", Status: "concluido"}}}
	rr := doReq(t, newTestHandler(svc).Routes(), http.MethodGet, "/emp-1/jobs/j1/progress", nil, bearerToken(t, "viewer"))
	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200 — %s", rr.Code, rr.Body.String())
	}
}

// garante que o pacote webhooks é usado (evitar import não utilizado em testes)
var _ = webhooks.EventSyncConcluido

// Manutenção operacional — não exercitada por estes testes.
func (m *mockSvc) ListarConsultasAtivas(context.Context) ([]ConsultaAtiva, error) { return nil, nil }
func (m *mockSvc) CancelarConsulta(context.Context, int32) (bool, error)          { return false, nil }
func (m *mockSvc) RefreshViewsGrupo(context.Context, string) ([]RefreshViewResultado, error) {
	return nil, nil
}
