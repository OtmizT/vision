package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"omie-sync-api/internal/audit"
	"omie-sync-api/internal/auth"
	"omie-sync-api/internal/dados"
	"omie-sync-api/internal/empresas"
	"omie-sync-api/internal/grupos"
	"omie-sync-api/internal/omie_config"
	"omie-sync-api/internal/permissoes"
	syncsvc "omie-sync-api/internal/sync"
	"omie-sync-api/internal/usuarios"
	"omie-sync-api/internal/webhooks"
)

const testSecret = "test-secret-minimo-32-caracteres-xpto"

// buildRouter monta o router completo com todos os mocks (sem banco).
func buildRouter(t *testing.T) http.Handler {
	t.Helper()

	jwtSvc := auth.NewJWTService(testSecret)

	// RepositÃ³rios nil â€” serÃ£o mocked via interfaces vazias apenas para construÃ§Ã£o
	auditRepo := &nullAuditRepo{}

	authSvc := &nullAuthSvc{}
	gruposSvc := grupos.NewService(&nullGruposRepo{}, nil)
	empresasSvc := empresas.NewService(&nullEmpresasRepo{})
	dispatcher := &nullDispatcher{}
	syncSvc := syncsvc.NewService(&nullSyncRepo{}, dispatcher, zerolog.Nop())
	usuariosSvc := usuarios.NewService(&nullUsuariosRepo{})
	permissoesSvc := permissoes.NewService(&nullPermissoesRepo{})
	omieConfigSvc := omie_config.NewService(&nullOmieConfigRepo{})

	return NewRouter(Dependencies{
		AuditRepo:         auditRepo,
		AuthHandler:       auth.NewHandler(authSvc, jwtSvc),
		GruposHandler:     grupos.NewHandler(gruposSvc, jwtSvc, nil),
		EmpresasHandler:   empresas.NewHandler(empresasSvc, jwtSvc, nil),
		SyncHandler:       syncsvc.NewHandler(syncSvc, jwtSvc, syncsvc.NewSSEHub()),
		UsuariosHandler:   usuarios.NewHandler(usuariosSvc, jwtSvc, nil),
		PermissoesHandler: permissoes.NewHandler(permissoesSvc, jwtSvc),
		DadosHandler:      dados.NewHandler(nil, jwtSvc),
		OmieConfigHandler: omie_config.NewHandler(omieConfigSvc, jwtSvc),
		Logger:            zerolog.Nop(),
	})
}

// --- Testes de integraÃ§Ã£o ---

func TestIntegration_HealthCheck(t *testing.T) {
	router := buildRouter(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("health: got %d want 200", rr.Code)
	}
	var body map[string]string
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["status"] != "ok" {
		t.Errorf("body: %s", rr.Body.String())
	}
}

func TestIntegration_UnauthenticatedRoutesReturn401(t *testing.T) {
	router := buildRouter(t)

	routes := []struct{ method, path string }{
		{http.MethodGet, "/admin/grupos"},
		{http.MethodPost, "/admin/grupos"},
		{http.MethodGet, "/admin/grupos/some-id"},
		{http.MethodGet, "/sync/emp-1/status"},
		{http.MethodGet, "/sync/emp-1/jobs"},
		{http.MethodGet, "/admin/permissoes/usuario/u1"},
	}

	for _, r := range routes {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(r.method, r.path, nil)
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: got %d want 401", r.method, r.path, rr.Code)
		}
	}
}

func TestIntegration_AuditMiddlewareRunsOnAllRoutes(t *testing.T) {
	logged := &logCapture{notify: make(chan struct{}, 10)}
	jwtSvc := auth.NewJWTService(testSecret)

	router := NewRouter(Dependencies{
		AuditRepo:         logged,
		AuthHandler:       auth.NewHandler(&nullAuthSvc{}, jwtSvc),
		GruposHandler:     grupos.NewHandler(grupos.NewService(&nullGruposRepo{}, nil), jwtSvc, nil),
		EmpresasHandler:   empresas.NewHandler(empresas.NewService(&nullEmpresasRepo{}), jwtSvc, nil),
		SyncHandler:       syncsvc.NewHandler(syncsvc.NewService(&nullSyncRepo{}, &nullDispatcher{}, zerolog.Nop()), jwtSvc, syncsvc.NewSSEHub()),
		UsuariosHandler:   usuarios.NewHandler(usuarios.NewService(&nullUsuariosRepo{}), jwtSvc, nil),
		PermissoesHandler: permissoes.NewHandler(permissoes.NewService(&nullPermissoesRepo{}), jwtSvc),
		DadosHandler:      dados.NewHandler(nil, jwtSvc),
		OmieConfigHandler: omie_config.NewHandler(omie_config.NewService(&nullOmieConfigRepo{}), jwtSvc),
		Logger:            zerolog.Nop(),
	})

	for _, path := range []string{"/health", "/auth/login"} {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		router.ServeHTTP(rr, req)
	}

	// Aguarda atÃ© 2 entradas de audit (uma por request) com timeout
	timeout := time.After(2 * time.Second)
	for logged.count < 2 {
		select {
		case <-logged.notify:
		case <-timeout:
			t.Fatalf("audit middleware registrou apenas %d entradas (esperava 2)", logged.count)
		}
	}
}

func TestIntegration_LoginEndpointReachable(t *testing.T) {
	router := buildRouter(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	router.ServeHTTP(rr, req)

	// Body vazio â†’ 422, mas a rota existe (nÃ£o Ã© 404)
	if rr.Code == http.StatusNotFound {
		t.Fatalf("/auth/login retornou 404 â€” rota nÃ£o registrada")
	}
}

func TestIntegration_AllAdminRoutesExist(t *testing.T) {
	router := buildRouter(t)
	tok, _ := auth.NewJWTService(testSecret).Generate("u1", "g1", "u@t.com", "admin_global")

	paths := []struct{ method, path string }{
		{http.MethodGet, "/admin/grupos"},
		{http.MethodPost, "/admin/grupos"},
		{http.MethodGet, "/admin/grupos/some-id"},
		{http.MethodPut, "/admin/grupos/some-id"},
		{http.MethodDelete, "/admin/grupos/some-id"},
		{http.MethodGet, "/admin/permissoes/usuario/u1"},
		{http.MethodGet, "/admin/permissoes/empresa/e1"},
	}

	for _, r := range paths {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(r.method, r.path, nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		router.ServeHTTP(rr, req)
		if rr.Code == http.StatusNotFound {
			t.Errorf("%s %s retornou 404 â€” rota nÃ£o registrada", r.method, r.path)
		}
	}
}

// --- Null implementations ---

type nullAuditRepo struct{ count int }

func (r *nullAuditRepo) Insert(_ context.Context, _ audit.LogEntry) error {
	r.count++
	return nil
}

type logCapture struct {
	count  int
	notify chan struct{}
}

func (r *logCapture) Insert(_ context.Context, _ audit.LogEntry) error {
	r.count++
	select {
	case r.notify <- struct{}{}:
	default:
	}
	return nil
}

type nullAuthSvc struct{}

func (s *nullAuthSvc) Login(_ context.Context, _, _ string) (*auth.LoginResponse, error) {
	return nil, nil
}
func (s *nullAuthSvc) SelectGrupo(_ context.Context, _, _ string) (*auth.LoginResponse, error) {
	return nil, nil
}
func (s *nullAuthSvc) TrocaGrupo(_ context.Context, _, _ string) (*auth.LoginResponse, error) {
	return nil, nil
}
func (s *nullAuthSvc) GetGrupos(_ context.Context, _ string) ([]auth.GrupoInfo, error) {
	return nil, nil
}
func (s *nullAuthSvc) Logout(_ context.Context, _ string) error { return nil }
func (s *nullAuthSvc) Refresh(_ context.Context, _ string) (*auth.LoginResponse, error) {
	return nil, nil
}
func (s *nullAuthSvc) Me(_ context.Context, _ string) (*auth.MeResponse, error) { return nil, nil }

type nullGruposRepo struct{}

func (r *nullGruposRepo) Insert(_ context.Context, _, _, _ string) (*grupos.Grupo, error) {
	return nil, nil
}
func (r *nullGruposRepo) GetByID(_ context.Context, _ string) (*grupos.Grupo, error) {
	return nil, nil
}
func (r *nullGruposRepo) GetBySlug(_ context.Context, _ string) (*grupos.Grupo, error) {
	return nil, nil
}
func (r *nullGruposRepo) List(_ context.Context, _, _ int32) ([]*grupos.Grupo, error) {
	return nil, nil
}
func (r *nullGruposRepo) Count(_ context.Context) (int64, error) { return 0, nil }
func (r *nullGruposRepo) Update(_ context.Context, _, _ string) (*grupos.Grupo, error) {
	return nil, nil
}
func (r *nullGruposRepo) SoftDelete(_ context.Context, _ string) error { return nil }
func (r *nullGruposRepo) CountEmpresasAtivas(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

type nullEmpresasRepo struct{}

func (r *nullEmpresasRepo) Insert(_ context.Context, _, _, _, _, _ string) (*empresas.Empresa, error) {
	return nil, nil
}
func (r *nullEmpresasRepo) GetByID(_ context.Context, _ string) (*empresas.Empresa, error) {
	return nil, nil
}
func (r *nullEmpresasRepo) List(_ context.Context, _ string, _, _ int32) ([]*empresas.Empresa, error) {
	return nil, nil
}
func (r *nullEmpresasRepo) Count(_ context.Context, _ string) (int64, error) { return 0, nil }
func (r *nullEmpresasRepo) Update(_ context.Context, _, _, _, _, _ string) (*empresas.Empresa, error) {
	return nil, nil
}
func (r *nullEmpresasRepo) MarkDeletando(_ context.Context, _ string) error { return nil }
func (r *nullEmpresasRepo) InsertDeletionQueue(_ context.Context, _ string, _ time.Time) error {
	return nil
}
func (r *nullEmpresasRepo) Reativar(_ context.Context, _ string) error            { return nil }
func (r *nullEmpresasRepo) CancelDeletionQueue(_ context.Context, _ string) error { return nil }
func (r *nullEmpresasRepo) ListPendingDeletions(_ context.Context) ([]empresas.PendingDeletion, error) {
	return nil, nil
}
func (r *nullEmpresasRepo) MarkDeletionExecuted(_ context.Context, _ string) error { return nil }

type nullSyncRepo struct{}

func (r *nullSyncRepo) InsertJob(_ context.Context, _, _, _ string) (*syncsvc.SyncJob, error) {
	return nil, nil
}
func (r *nullSyncRepo) GetJobByID(_ context.Context, _ string) (*syncsvc.SyncJob, error) {
	return nil, nil
}
func (r *nullSyncRepo) ListJobs(_ context.Context, _ string, _, _ int32) ([]*syncsvc.SyncJob, error) {
	return nil, nil
}
func (r *nullSyncRepo) CountJobs(_ context.Context, _ string) (int64, error) { return 0, nil }
func (r *nullSyncRepo) UpdateJobStatus(_ context.Context, _, _, _ string, _, _ *time.Time) (*syncsvc.SyncJob, error) {
	return nil, nil
}
func (r *nullSyncRepo) GetControl(_ context.Context, _ string) (*syncsvc.SyncControl, error) {
	return nil, nil
}
func (r *nullSyncRepo) UltimoSyncDoGrupo(_ context.Context, _ string) (*time.Time, error) {
	return nil, nil
}
func (r *nullSyncRepo) UpsertControl(_ context.Context, _ string, _ bool, _, _ int, _, _ *time.Time) (*syncsvc.SyncControl, error) {
	return nil, nil
}
func (r *nullSyncRepo) UpdateControlAfterRun(_ context.Context, _, _ string) error     { return nil }
func (r *nullSyncRepo) AdvanceScheduleOnDispatch(_ context.Context, _, _ string) error { return nil }
func (r *nullSyncRepo) GetJobProgress(_ context.Context, _ string) ([]*syncsvc.SyncJobProgress, error) {
	return nil, nil
}
func (r *nullSyncRepo) GetExecutorConfigs(_ context.Context, _ string) ([]*syncsvc.EmpresaExecutorConfig, error) {
	return nil, nil
}
func (r *nullSyncRepo) UpsertExecutorConfig(_ context.Context, _, _ string, _ bool, _ *string, _ string) (*syncsvc.EmpresaExecutorConfig, error) {
	return nil, nil
}
func (r *nullSyncRepo) GetEnabledExecutors(_ context.Context, _ string) (map[string]bool, error) {
	return make(map[string]bool), nil
}
func (r *nullSyncRepo) GetJobAtivo(_ context.Context, _ string) (*syncsvc.JobAtivoResult, error) {
	return nil, nil
}
func (r *nullSyncRepo) MarkStaleJobs(_ context.Context) (int64, error)       { return 0, nil }
func (r *nullSyncRepo) UpdateJobHeartbeat(_ context.Context, _ string) error { return nil }
func (r *nullSyncRepo) GetJobsOverview(_ context.Context) ([]syncsvc.JobStatusCount, error) {
	return nil, nil
}
func (r *nullSyncRepo) GetJobsAtivos(_ context.Context) ([]syncsvc.JobAtivoRow, error) {
	return nil, nil
}
func (r *nullSyncRepo) ListEmpresasSyncGeral(_ context.Context) ([]syncsvc.EmpresaSyncRow, error) {
	return nil, nil
}
func (r *nullSyncRepo) CancelarJob(_ context.Context, _ string) error                { return nil }
func (r *nullSyncRepo) InsertJobPage(_ context.Context, _, _ string, _, _ int) error { return nil }
func (r *nullSyncRepo) GetPendingPages(_ context.Context, _ string, _ int) ([]syncsvc.JobPage, error) {
	return nil, nil
}
func (r *nullSyncRepo) CountPendingPages(_ context.Context, _ string) (int64, error) { return 0, nil }
func (r *nullSyncRepo) ClaimPageForProcessing(_ context.Context, _ string) (*syncsvc.JobPage, error) {
	return nil, nil
}
func (r *nullSyncRepo) MarkPageConcluido(_ context.Context, _ string, _ int) error { return nil }
func (r *nullSyncRepo) MarkPageErro(_ context.Context, _ string, _ string, _ time.Time) error {
	return nil
}
func (r *nullSyncRepo) MarkPageCancelado(_ context.Context, _ string) error         { return nil }
func (r *nullSyncRepo) GetDLQPages(_ context.Context) ([]syncsvc.DLQPageRow, error) { return nil, nil }
func (r *nullSyncRepo) RetryDLQPage(_ context.Context, _ string) error              { return nil }
func (r *nullSyncRepo) GetPagesByJob(_ context.Context, _ string) ([]syncsvc.PageRow, error) {
	return nil, nil
}
func (r *nullSyncRepo) GetLatestJobIDByEmpresa(_ context.Context, _ string) (string, error) {
	return "", nil
}

type nullDispatcher struct{}

func (d *nullDispatcher) Dispatch(_ string, _ webhooks.Event) {}

type nullUsuariosRepo struct{}

func (r *nullUsuariosRepo) Insert(_ context.Context, _, _, _, _, _ string) (*usuarios.Usuario, error) {
	return nil, nil
}
func (r *nullUsuariosRepo) GetByID(_ context.Context, _ string) (*usuarios.Usuario, error) {
	return nil, nil
}
func (r *nullUsuariosRepo) List(_ context.Context, _ string, _, _ int32) ([]*usuarios.Usuario, error) {
	return nil, nil
}
func (r *nullUsuariosRepo) Count(_ context.Context, _ string) (int64, error) { return 0, nil }
func (r *nullUsuariosRepo) Update(_ context.Context, _, _, _, _ string, _ bool) (*usuarios.Usuario, error) {
	return nil, nil
}
func (r *nullUsuariosRepo) GetByEmail(_ context.Context, _ string) (*usuarios.Usuario, error) {
	return nil, errors.New("not found")
}
func (r *nullUsuariosRepo) HasGrupoVinculo(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}
func (r *nullUsuariosRepo) UpdatePassword(_ context.Context, _, _ string) error        { return nil }
func (r *nullUsuariosRepo) SoftDelete(_ context.Context, _ string) error               { return nil }
func (r *nullUsuariosRepo) InsertGrupoVinculo(_ context.Context, _, _, _ string) error { return nil }

type nullPermissoesRepo struct{}

func (r *nullPermissoesRepo) Grant(_ context.Context, _, _, _, _ string) (*permissoes.Permissao, error) {
	return nil, nil
}
func (r *nullPermissoesRepo) Revoke(_ context.Context, _, _, _, _ string) error { return nil }
func (r *nullPermissoesRepo) ListByUsuario(_ context.Context, _ string) ([]*permissoes.Permissao, error) {
	return nil, nil
}
func (r *nullPermissoesRepo) ListByEmpresa(_ context.Context, _ string) ([]*permissoes.Permissao, error) {
	return nil, nil
}
func (r *nullPermissoesRepo) Has(_ context.Context, _, _, _, _ string) (bool, error) {
	return false, nil
}

type nullOmieConfigRepo struct{}

func (r *nullOmieConfigRepo) GetAll(_ context.Context) ([]*omie_config.EndpointConfig, error) {
	return nil, nil
}
func (r *nullOmieConfigRepo) GetByModulo(_ context.Context, _ string) (*omie_config.EndpointConfig, error) {
	return nil, nil
}
func (r *nullOmieConfigRepo) Update(_ context.Context, _ string, _ omie_config.UpdateRequest, _ string) (*omie_config.EndpointConfig, error) {
	return nil, nil
}

// Manutenção operacional — não exercitada por estes testes.
func (n *nullSyncRepo) ConsultasAtivas(context.Context) ([]syncsvc.ConsultaAtiva, error) {
	return nil, nil
}
func (n *nullSyncRepo) CancelarConsulta(context.Context, int32) (bool, error)     { return false, nil }
func (n *nullSyncRepo) SchemaDoGrupo(context.Context, string) (string, error)     { return "", nil }
func (n *nullSyncRepo) RefreshView(context.Context, string, string) (bool, error) { return false, nil }

/*
Fase A — as portas dos fundos que estavam abertas em produção.

Os três casos abaixo passavam antes: o primeiro criava um grupo, os outros dois
liam e escreviam no cliente errado. São testes de roteamento — checam que o
middleware está montado, não a regra dele, que vive em auth/middleware_test.go.
*/
func TestIntegration_IsolamentoEntreClientes(t *testing.T) {
	router := buildRouter(t)
	jwtSvc := auth.NewJWTService(testSecret)

	const grupoA = "11111111-1111-1111-1111-111111111111"
	const grupoB = "22222222-2222-2222-2222-222222222222"

	viewer, _ := jwtSvc.Generate("u-v", grupoA, "v@a.com", "viewer")
	adminA, _ := jwtSvc.Generate("u-a", grupoA, "a@a.com", "admin_grupo")

	casos := []struct {
		nome         string
		method, path string
		token        string
		esperado     int
	}{
		// Criar grupo estava fora de qualquer bloco de papel.
		{"viewer não cria grupo", http.MethodPost, "/admin/grupos", viewer, http.StatusForbidden},
		{"admin de grupo não cria grupo", http.MethodPost, "/admin/grupos", adminA, http.StatusForbidden},

		// A rota de usuários só verificava papel, nunca o grupo da URL.
		{"usuários de outro cliente", http.MethodGet, "/admin/grupos/" + grupoB + "/usuarios", adminA, http.StatusForbidden},
		{"empresas de outro cliente", http.MethodGet, "/admin/grupos/" + grupoB + "/empresas", adminA, http.StatusForbidden},
		{"SQL Explorer em outro cliente", http.MethodPost, "/admin/grupos/" + grupoB + "/query", adminA, http.StatusForbidden},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(c.method, c.path, nil)
			req.Header.Set("Authorization", "Bearer "+c.token)
			router.ServeHTTP(rr, req)
			if rr.Code != c.esperado {
				t.Fatalf("got %d, want %d", rr.Code, c.esperado)
			}
		})
	}
}

// O fechamento não pode ter trancado quem é de casa.
func TestIntegration_AcessoLegitimoContinuaPassando(t *testing.T) {
	router := buildRouter(t)
	jwtSvc := auth.NewJWTService(testSecret)

	const grupoA = "11111111-1111-1111-1111-111111111111"
	adminA, _ := jwtSvc.Generate("u-a", grupoA, "a@a.com", "admin_grupo")
	global, _ := jwtSvc.Generate("u-g", "", "g@p.com", "admin_global")

	casos := []struct {
		nome         string
		method, path string
		token        string
	}{
		{"admin do grupo lista os próprios usuários", http.MethodGet, "/admin/grupos/" + grupoA + "/usuarios", adminA},
		{"admin do grupo lista as próprias empresas", http.MethodGet, "/admin/grupos/" + grupoA + "/empresas", adminA},
		{"admin global cria grupo", http.MethodPost, "/admin/grupos", global},
		{"admin global entra em qualquer grupo", http.MethodGet, "/admin/grupos/" + grupoA + "/usuarios", global},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(c.method, c.path, nil)
			req.Header.Set("Authorization", "Bearer "+c.token)
			router.ServeHTTP(rr, req)
			if rr.Code == http.StatusForbidden || rr.Code == http.StatusUnauthorized || rr.Code == http.StatusNotFound {
				t.Fatalf("acesso legítimo barrado: got %d", rr.Code)
			}
		})
	}
}

// capturaEntradas guarda as LogEntry inteiras, e não só a contagem.
type capturaEntradas struct {
	ch chan audit.LogEntry
}

func (c *capturaEntradas) Insert(_ context.Context, e audit.LogEntry) error {
	c.ch <- e
	return nil
}

/*
O autor chegando ao log através do router de verdade.

Os testes de unidade cobrem as duas metades separadas — o middleware de
auditoria deixando o Ator no contexto, e o RequireAuth preenchendo. Só que o
defeito original era exatamente a junção: a autenticação roda dentro de cada
subrouter montado, depois da auditoria, e a escrita precisa atravessar o
r.WithContext de volta. Isso só a cadeia inteira mostra.
*/
func TestIntegration_AuditoriaRegistraQuemFez(t *testing.T) {
	cap := &capturaEntradas{ch: make(chan audit.LogEntry, 8)}
	jwtSvc := auth.NewJWTService(testSecret)

	router := NewRouter(Dependencies{
		AuditRepo:         cap,
		AuthHandler:       auth.NewHandler(&nullAuthSvc{}, jwtSvc),
		GruposHandler:     grupos.NewHandler(grupos.NewService(&nullGruposRepo{}, nil), jwtSvc, nil),
		EmpresasHandler:   empresas.NewHandler(empresas.NewService(&nullEmpresasRepo{}), jwtSvc, nil),
		SyncHandler:       syncsvc.NewHandler(syncsvc.NewService(&nullSyncRepo{}, &nullDispatcher{}, zerolog.Nop()), jwtSvc, syncsvc.NewSSEHub()),
		UsuariosHandler:   usuarios.NewHandler(usuarios.NewService(&nullUsuariosRepo{}), jwtSvc, nil),
		PermissoesHandler: permissoes.NewHandler(permissoes.NewService(&nullPermissoesRepo{}), jwtSvc),
		DadosHandler:      dados.NewHandler(nil, jwtSvc),
		OmieConfigHandler: omie_config.NewHandler(omie_config.NewService(&nullOmieConfigRepo{}), jwtSvc),
		Logger:            zerolog.Nop(),
	})

	tok, _ := jwtSvc.Generate("u-77", "g-1", "ana@alpha.com", "admin_global")
	req := httptest.NewRequest(http.MethodGet, "/admin/grupos", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	router.ServeHTTP(httptest.NewRecorder(), req)

	select {
	case e := <-cap.ch:
		if e.UserID != "u-77" || e.UserEmail != "ana@alpha.com" || e.Role != "admin_global" {
			t.Fatalf("trilha sem autor: %+v", e)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("nenhuma entrada de auditoria gravada")
	}
}

// Uma requisição barrada é justamente a que precisa de autor na trilha.
func TestIntegration_AuditoriaRegistraAutorDeAcessoNegado(t *testing.T) {
	cap := &capturaEntradas{ch: make(chan audit.LogEntry, 8)}
	jwtSvc := auth.NewJWTService(testSecret)

	router := NewRouter(Dependencies{
		AuditRepo:         cap,
		AuthHandler:       auth.NewHandler(&nullAuthSvc{}, jwtSvc),
		GruposHandler:     grupos.NewHandler(grupos.NewService(&nullGruposRepo{}, nil), jwtSvc, nil),
		EmpresasHandler:   empresas.NewHandler(empresas.NewService(&nullEmpresasRepo{}), jwtSvc, nil),
		SyncHandler:       syncsvc.NewHandler(syncsvc.NewService(&nullSyncRepo{}, &nullDispatcher{}, zerolog.Nop()), jwtSvc, syncsvc.NewSSEHub()),
		UsuariosHandler:   usuarios.NewHandler(usuarios.NewService(&nullUsuariosRepo{}), jwtSvc, nil),
		PermissoesHandler: permissoes.NewHandler(permissoes.NewService(&nullPermissoesRepo{}), jwtSvc),
		DadosHandler:      dados.NewHandler(nil, jwtSvc),
		OmieConfigHandler: omie_config.NewHandler(omie_config.NewService(&nullOmieConfigRepo{}), jwtSvc),
		Logger:            zerolog.Nop(),
	})

	tok, _ := jwtSvc.Generate("u-88", "11111111-1111-1111-1111-111111111111", "mal@a.com", "admin_grupo")
	req := httptest.NewRequest(http.MethodGet, "/admin/grupos/22222222-2222-2222-2222-222222222222/usuarios", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("got %d, want 403", rr.Code)
	}
	select {
	case e := <-cap.ch:
		if e.UserID != "u-88" || e.StatusCode != http.StatusForbidden {
			t.Fatalf("tentativa negada sem autor: %+v", e)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("nenhuma entrada de auditoria gravada")
	}
}
