package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"omie-sync-api/internal/etl/progress"
	"omie-sync-api/internal/omie"
	"omie-sync-api/internal/omie_config"
	syncsvc "omie-sync-api/internal/sync"
	"omie-sync-api/internal/webhooks"
)

// --- mocks ---

type mockSyncRepo struct {
	job            *syncsvc.SyncJob
	updatedStatus  string
	updatedErro    string
	afterRunCalled bool
}

func (m *mockSyncRepo) InsertJob(_ context.Context, empresaID, tipo, executor string) (*syncsvc.SyncJob, error) {
	return &syncsvc.SyncJob{ID: "j1", EmpresaID: empresaID, Tipo: tipo, Status: "pendente", Executor: executor}, nil
}
func (m *mockSyncRepo) GetJobByID(_ context.Context, _ string) (*syncsvc.SyncJob, error) {
	return m.job, nil
}
func (m *mockSyncRepo) ListJobs(_ context.Context, _ string, _, _ int32) ([]*syncsvc.SyncJob, error) {
	return nil, nil
}
func (m *mockSyncRepo) CountJobs(_ context.Context, _ string) (int64, error) { return 0, nil }
func (m *mockSyncRepo) UpdateJobStatus(_ context.Context, _ string, status, erro string, _, _ *time.Time) (*syncsvc.SyncJob, error) {
	m.updatedStatus = status
	m.updatedErro = erro
	return &syncsvc.SyncJob{Status: status}, nil
}
func (m *mockSyncRepo) GetControl(_ context.Context, _ string) (*syncsvc.SyncControl, error) {
	return nil, nil
}
func (m *mockSyncRepo) UltimoSyncDoGrupo(_ context.Context, _ string) (*time.Time, error) {
	return nil, nil
}
func (m *mockSyncRepo) UpsertControl(_ context.Context, _ string, _ bool, _, _ int, _, _ *time.Time) (*syncsvc.SyncControl, error) {
	return nil, nil
}
func (m *mockSyncRepo) UpdateControlAfterRun(_ context.Context, _, _ string) error {
	m.afterRunCalled = true
	return nil
}
func (m *mockSyncRepo) AdvanceScheduleOnDispatch(_ context.Context, _, _ string) error {
	return nil
}
func (m *mockSyncRepo) GetJobProgress(_ context.Context, _ string) ([]*syncsvc.SyncJobProgress, error) {
	return nil, nil
}
func (m *mockSyncRepo) MarkStaleJobs(_ context.Context) (int64, error)       { return 0, nil }
func (m *mockSyncRepo) UpdateJobHeartbeat(_ context.Context, _ string) error { return nil }
func (m *mockSyncRepo) GetJobsOverview(_ context.Context) ([]syncsvc.JobStatusCount, error) {
	return nil, nil
}
func (m *mockSyncRepo) GetJobsAtivos(_ context.Context) ([]syncsvc.JobAtivoRow, error) {
	return nil, nil
}
func (m *mockSyncRepo) ListEmpresasSyncGeral(_ context.Context) ([]syncsvc.EmpresaSyncRow, error) {
	return nil, nil
}
func (m *mockSyncRepo) CancelarJob(_ context.Context, _ string) error                { return nil }
func (m *mockSyncRepo) InsertJobPage(_ context.Context, _, _ string, _, _ int) error { return nil }
func (m *mockSyncRepo) GetPendingPages(_ context.Context, _ string, _ int) ([]syncsvc.JobPage, error) {
	return nil, nil
}
func (m *mockSyncRepo) CountPendingPages(_ context.Context, _ string) (int64, error) { return 0, nil }
func (m *mockSyncRepo) ClaimPageForProcessing(_ context.Context, _ string) (*syncsvc.JobPage, error) {
	return nil, nil
}
func (m *mockSyncRepo) MarkPageConcluido(_ context.Context, _ string, _ int) error { return nil }
func (m *mockSyncRepo) MarkPageErro(_ context.Context, _ string, _ string, _ time.Time) error {
	return nil
}
func (m *mockSyncRepo) MarkPageCancelado(_ context.Context, _ string) error         { return nil }
func (m *mockSyncRepo) GetDLQPages(_ context.Context) ([]syncsvc.DLQPageRow, error) { return nil, nil }
func (m *mockSyncRepo) RetryDLQPage(_ context.Context, _ string) error              { return nil }
func (m *mockSyncRepo) GetPagesByJob(_ context.Context, _ string) ([]syncsvc.PageRow, error) {
	return nil, nil
}
func (m *mockSyncRepo) GetLatestJobIDByEmpresa(_ context.Context, _ string) (string, error) {
	return "", nil
}
func (m *mockSyncRepo) GetExecutorConfigs(_ context.Context, _ string) ([]*syncsvc.EmpresaExecutorConfig, error) {
	return nil, nil
}
func (m *mockSyncRepo) UpsertExecutorConfig(_ context.Context, _, _ string, _ bool, _ *string, _ string) (*syncsvc.EmpresaExecutorConfig, error) {
	return nil, nil
}
func (m *mockSyncRepo) GetEnabledExecutors(_ context.Context, _ string) (map[string]bool, error) {
	return make(map[string]bool), nil
}
func (m *mockSyncRepo) GetJobAtivo(_ context.Context, _ string) (*syncsvc.JobAtivoResult, error) {
	return nil, nil
}

type mockOmieConfig struct{}

func (m *mockOmieConfig) GetAll(_ context.Context) ([]*omie_config.EndpointConfig, error) {
	return nil, nil
}
func (m *mockOmieConfig) GetByModulo(_ context.Context, _ string) (*omie_config.EndpointConfig, error) {
	return nil, nil
}
func (m *mockOmieConfig) Update(_ context.Context, _ string, _ omie_config.UpdateRequest, _ string) (*omie_config.EndpointConfig, error) {
	return nil, nil
}
func (m *mockOmieConfig) GetCached(_ context.Context, modulo string) (*omie_config.EndpointConfig, error) {
	return &omie_config.EndpointConfig{Modulo: modulo, Ativo: true}, nil
}
func (m *mockOmieConfig) RefreshCache(_ context.Context) error { return nil }

type mockFetcher struct {
	creds *EmpresaCredentials
	err   error
}

func (m *mockFetcher) GetActiveCredentials(_ context.Context, _ string) (*EmpresaCredentials, error) {
	return m.creds, m.err
}
func (m *mockFetcher) PausarEmpresa(_ context.Context, _ string) error { return nil }

type mockDispatcher struct {
	events []webhooks.Event
}

func (m *mockDispatcher) Dispatch(_ string, e webhooks.Event) {
	m.events = append(m.events, e)
}

type mockExecutor struct {
	nome   string
	err    error
	called bool
}

func (m *mockExecutor) Nome() string { return m.nome }
func (m *mockExecutor) Execute(_ context.Context, _ *omie.Client, _ string, _ SyncOptions, _ string, _ progress.Reporter, _ *omie_config.EndpointConfig) error {
	m.called = true
	return m.err
}
func (m *mockExecutor) ExecutePage(_ context.Context, _ *omie.Client, _ string, _ SyncOptions, _ int, _ *omie_config.EndpointConfig) (int, error) {
	return 0, nil
}

// --- testes ---

func newTestWorker(repo *mockSyncRepo, fetcher *mockFetcher, dispatcher *mockDispatcher, execs []Executor) *Worker {
	return NewWorker(repo, fetcher, execs, dispatcher, &progress.NoopReporter{}, &mockOmieConfig{}, syncsvc.NewSSEHub(), nil, zerolog.Nop())
}

func defaultCreds() *EmpresaCredentials {
	return &EmpresaCredentials{
		ID: "e1", GrupoID: "g1", AppKey: "k", AppSecret: "s", Schema: "grupo_test",
	}
}

func TestWorker_ProcessJob_Success(t *testing.T) {
	repo := &mockSyncRepo{job: &syncsvc.SyncJob{ID: "j1", EmpresaID: "e1", Tipo: "manual"}}
	fetcher := &mockFetcher{creds: defaultCreds()}
	dispatcher := &mockDispatcher{}
	exec := &mockExecutor{nome: "clientes"}

	w := newTestWorker(repo, fetcher, dispatcher, []Executor{exec})
	err := w.ProcessJob(context.Background(), "j1")

	if err != nil {
		t.Fatalf("ProcessJob: %v", err)
	}
	if !exec.called {
		t.Error("executor não foi chamado")
	}
	if repo.updatedStatus != "concluido" {
		t.Errorf("status: got %q want concluido", repo.updatedStatus)
	}
	if !repo.afterRunCalled {
		t.Error("UpdateControlAfterRun não foi chamado")
	}
	if len(dispatcher.events) == 0 {
		t.Error("webhook não foi disparado")
	}
	if dispatcher.events[len(dispatcher.events)-1].Tipo != webhooks.EventSyncConcluido {
		t.Errorf("evento: got %q want %q", dispatcher.events[0].Tipo, webhooks.EventSyncConcluido)
	}
}

func TestWorker_ProcessJob_ExecutorError(t *testing.T) {
	repo := &mockSyncRepo{job: &syncsvc.SyncJob{ID: "j1", EmpresaID: "e1", Tipo: "manual"}}
	fetcher := &mockFetcher{creds: defaultCreds()}
	dispatcher := &mockDispatcher{}
	exec := &mockExecutor{nome: "clientes", err: errors.New("timeout da API")}

	w := newTestWorker(repo, fetcher, dispatcher, []Executor{exec})
	w.ProcessJob(context.Background(), "j1")

	if repo.updatedStatus != "erro" {
		t.Errorf("status: got %q want erro", repo.updatedStatus)
	}
	// Webhook de falha deve ser disparado
	found := false
	for _, e := range dispatcher.events {
		if e.Tipo == webhooks.EventSyncFalhou {
			found = true
		}
	}
	if !found {
		t.Error("webhook sync.falhou não disparado")
	}
}

func TestWorker_ProcessJob_CredencialInvalida(t *testing.T) {
	repo := &mockSyncRepo{job: &syncsvc.SyncJob{ID: "j1", EmpresaID: "e1", Tipo: "manual"}}
	fetcher := &mockFetcher{creds: defaultCreds()}
	dispatcher := &mockDispatcher{}
	exec := &mockExecutor{
		nome: "clientes",
		err:  omie.OmieError{FaultCode: omie.ErrCodeCredencialInvalida, FaultString: "Credencial inválida"},
	}

	w := newTestWorker(repo, fetcher, dispatcher, []Executor{exec})
	w.ProcessJob(context.Background(), "j1")

	if repo.updatedStatus != "erro" {
		t.Errorf("status: got %q want erro", repo.updatedStatus)
	}
	// Deve disparar empresa.pausada
	found := false
	for _, e := range dispatcher.events {
		if e.Tipo == webhooks.EventEmpresaPausada {
			found = true
		}
	}
	if !found {
		t.Error("webhook empresa.pausada não disparado para credencial inválida")
	}
}

func TestWorker_ProcessJob_MultipleExecutors(t *testing.T) {
	repo := &mockSyncRepo{job: &syncsvc.SyncJob{ID: "j1", EmpresaID: "e1", Tipo: "full"}}
	fetcher := &mockFetcher{creds: defaultCreds()}
	dispatcher := &mockDispatcher{}

	execs := []Executor{
		&mockExecutor{nome: "clientes"},
		&mockExecutor{nome: "categorias"},
		&mockExecutor{nome: "contas_pagar"},
	}

	w := newTestWorker(repo, fetcher, dispatcher, execs)
	w.ProcessJob(context.Background(), "j1")

	for _, e := range execs {
		if !e.(*mockExecutor).called {
			t.Errorf("executor %q não foi chamado", e.(*mockExecutor).nome)
		}
	}
	if repo.updatedStatus != "concluido" {
		t.Errorf("status: got %q want concluido", repo.updatedStatus)
	}
}

// Manutenção operacional — não exercitada por estes testes.
func (m *mockSyncRepo) ConsultasAtivas(context.Context) ([]syncsvc.ConsultaAtiva, error) {
	return nil, nil
}
func (m *mockSyncRepo) CancelarConsulta(context.Context, int32) (bool, error)     { return false, nil }
func (m *mockSyncRepo) SchemaDoGrupo(context.Context, string) (string, error)     { return "", nil }
func (m *mockSyncRepo) RefreshView(context.Context, string, string) (bool, error) { return false, nil }
