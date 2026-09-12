package sync

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"omie-sync-api/internal/apperror"
	"omie-sync-api/internal/webhooks"
)

// --- mocks ---

type mockRepo struct {
	job      *SyncJob
	jobs     []*SyncJob
	control  *SyncControl
	total    int64
	progress []*SyncJobProgress
	err      error
}

func (m *mockRepo) InsertJob(_ context.Context, empresaID, tipo, executor string) (*SyncJob, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &SyncJob{ID: "job-1", EmpresaID: empresaID, Tipo: tipo, Status: "pendente", Executor: executor}, nil
}
func (m *mockRepo) GetJobByID(_ context.Context, _ string) (*SyncJob, error) {
	return m.job, m.err
}
func (m *mockRepo) ListJobs(_ context.Context, _ string, _, _ int32) ([]*SyncJob, error) {
	return m.jobs, m.err
}
func (m *mockRepo) CountJobs(_ context.Context, _ string) (int64, error) {
	return m.total, m.err
}
func (m *mockRepo) UpdateJobStatus(_ context.Context, id, status, erro string, _, _ *time.Time) (*SyncJob, error) {
	return &SyncJob{ID: id, Status: status, Erro: erro}, m.err
}
func (m *mockRepo) UltimoSyncDoGrupo(_ context.Context, _ string) (*time.Time, error) {
	return nil, nil
}
func (m *mockRepo) GetControl(_ context.Context, _ string) (*SyncControl, error) {
	if m.control == nil {
		return nil, errors.New("não encontrado")
	}
	return m.control, nil
}
func (m *mockRepo) UpsertControl(_ context.Context, empresaID string, ativo bool, intervaloIncrementalMin, intervaloFullDias int, _, _ *time.Time) (*SyncControl, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &SyncControl{EmpresaID: empresaID, Ativo: ativo, IntervaloIncrementalMin: intervaloIncrementalMin, IntervaloFullDias: intervaloFullDias}, nil
}
func (m *mockRepo) UpdateControlAfterRun(_ context.Context, _, _ string) error     { return m.err }
func (m *mockRepo) AdvanceScheduleOnDispatch(_ context.Context, _, _ string) error { return m.err }
func (m *mockRepo) GetJobProgress(_ context.Context, _ string) ([]*SyncJobProgress, error) {
	return m.progress, m.err
}
func (m *mockRepo) GetExecutorConfigs(_ context.Context, _ string) ([]*EmpresaExecutorConfig, error) {
	return nil, nil
}
func (m *mockRepo) UpsertExecutorConfig(_ context.Context, _, _ string, _ bool, _ *string, _ string) (*EmpresaExecutorConfig, error) {
	return nil, nil
}
func (m *mockRepo) GetEnabledExecutors(_ context.Context, _ string) (map[string]bool, error) {
	return make(map[string]bool), nil
}
func (m *mockRepo) GetJobAtivo(_ context.Context, _ string) (*JobAtivoResult, error) {
	if m.job != nil && (m.job.Status == "rodando" || m.job.Status == "pendente") {
		return &JobAtivoResult{
			Total:      1,
			ID:         m.job.ID,
			Tipo:       m.job.Tipo,
			Status:     m.job.Status,
			IniciadoAt: m.job.IniciadoAt,
		}, nil
	}
	return nil, nil
}
func (m *mockRepo) MarkStaleJobs(_ context.Context) (int64, error)              { return 0, nil }
func (m *mockRepo) UpdateJobHeartbeat(_ context.Context, _ string) error        { return nil }
func (m *mockRepo) GetJobsOverview(_ context.Context) ([]JobStatusCount, error) { return nil, nil }
func (m *mockRepo) GetJobsAtivos(_ context.Context) ([]JobAtivoRow, error)      { return nil, nil }
func (m *mockRepo) ListEmpresasSyncGeral(_ context.Context) ([]EmpresaSyncRow, error) {
	return nil, nil
}
func (m *mockRepo) CancelarJob(_ context.Context, _ string) error                { return nil }
func (m *mockRepo) InsertJobPage(_ context.Context, _, _ string, _, _ int) error { return nil }
func (m *mockRepo) GetPendingPages(_ context.Context, _ string, _ int) ([]JobPage, error) {
	return nil, nil
}
func (m *mockRepo) CountPendingPages(_ context.Context, _ string) (int64, error) { return 0, nil }
func (m *mockRepo) ClaimPageForProcessing(_ context.Context, _ string) (*JobPage, error) {
	return nil, nil
}
func (m *mockRepo) MarkPageConcluido(_ context.Context, _ string, _ int) error            { return nil }
func (m *mockRepo) MarkPageErro(_ context.Context, _ string, _ string, _ time.Time) error { return nil }
func (m *mockRepo) MarkPageCancelado(_ context.Context, _ string) error                   { return nil }
func (m *mockRepo) GetDLQPages(_ context.Context) ([]DLQPageRow, error)                   { return nil, nil }
func (m *mockRepo) RetryDLQPage(_ context.Context, _ string) error                        { return nil }
func (m *mockRepo) GetPagesByJob(_ context.Context, _ string) ([]PageRow, error)          { return nil, nil }
func (m *mockRepo) GetLatestJobIDByEmpresa(_ context.Context, _ string) (string, error) {
	return "", nil
}

type mockDispatcher struct {
	dispatched []webhooks.Event
}

func (m *mockDispatcher) Dispatch(_ string, e webhooks.Event) {
	m.dispatched = append(m.dispatched, e)
}

// --- testes ---

func TestService_GetStatus_SemControl(t *testing.T) {
	svc := NewService(&mockRepo{jobs: []*SyncJob{}}, &mockDispatcher{}, zerolog.Nop())

	status, err := svc.GetStatus(context.Background(), "emp-1")
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if status.Control != nil {
		t.Error("control deveria ser nil")
	}
	if status.UltimoJob != nil {
		t.Error("ultimo_job deveria ser nil")
	}
}

func TestService_GetStatus_ComUltimoJob(t *testing.T) {
	repo := &mockRepo{
		control: &SyncControl{EmpresaID: "emp-1", Ativo: true},
		jobs:    []*SyncJob{{ID: "j1", Status: "concluido"}},
	}
	svc := NewService(repo, &mockDispatcher{}, zerolog.Nop())

	status, err := svc.GetStatus(context.Background(), "emp-1")
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if status.UltimoJob == nil || status.UltimoJob.ID != "j1" {
		t.Error("ultimo_job incorreto")
	}
}

func TestService_ForcarSync_Success(t *testing.T) {
	d := &mockDispatcher{}
	svc := NewService(&mockRepo{}, d, zerolog.Nop())

	job, err := svc.ForcarSync(context.Background(), "grp-1", "emp-1", ForcarSyncRequest{Tipo: "full"})
	if err != nil {
		t.Fatalf("ForcarSync: %v", err)
	}
	if job.Tipo != "full" {
		t.Errorf("tipo: got %q want full", job.Tipo)
	}
	// Webhook é disparado pelo worker após o job concluir, não pelo service.
	// ForcarSync apenas cria o job e dispara o worker em goroutine.
	if job.Status != "pendente" {
		t.Errorf("job deve iniciar como pendente, got %q", job.Status)
	}
}

func TestService_ForcarSync_TipoDefault(t *testing.T) {
	svc := NewService(&mockRepo{}, &mockDispatcher{}, zerolog.Nop())

	job, err := svc.ForcarSync(context.Background(), "g", "e", ForcarSyncRequest{})
	if err != nil {
		t.Fatalf("ForcarSync: %v", err)
	}
	if job.Tipo != "manual" {
		t.Errorf("tipo default: got %q want manual", job.Tipo)
	}
}

func TestService_Configurar_Success(t *testing.T) {
	svc := NewService(&mockRepo{}, &mockDispatcher{}, zerolog.Nop())

	ctrl, err := svc.Configurar(context.Background(), "emp-1", ConfigurarRequest{
		Ativo:                   true,
		IntervaloIncrementalMin: 60,
		IntervaloFullDias:       7,
	})
	if err != nil {
		t.Fatalf("Configurar: %v", err)
	}
	if ctrl.IntervaloIncrementalMin != 60 {
		t.Errorf("intervalo_incremental_min: got %d want 60", ctrl.IntervaloIncrementalMin)
	}
	if ctrl.IntervaloFullDias != 7 {
		t.Errorf("intervalo_full_dias: got %d want 7", ctrl.IntervaloFullDias)
	}
}

func TestService_Configurar_ValoresInvalidos(t *testing.T) {
	svc := NewService(&mockRepo{}, &mockDispatcher{}, zerolog.Nop())

	tests := []struct {
		name string
		req  ConfigurarRequest
	}{
		{"incremental invalido", ConfigurarRequest{Ativo: true, IntervaloIncrementalMin: 30, IntervaloFullDias: 7}},
		{"full invalido", ConfigurarRequest{Ativo: true, IntervaloIncrementalMin: 60, IntervaloFullDias: 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Configurar(context.Background(), "emp-1", tt.req)
			if err == nil {
				t.Fatal("esperava erro")
			}
			ae, ok := apperror.IsAppError(err)
			if !ok || ae.Code != 422 {
				t.Errorf("esperava 422, got %v", err)
			}
		})
	}
}

func TestService_ListJobs(t *testing.T) {
	repo := &mockRepo{
		jobs:  []*SyncJob{{ID: "j1"}, {ID: "j2"}},
		total: 2,
	}
	svc := NewService(repo, &mockDispatcher{}, zerolog.Nop())

	jobs, total, err := svc.ListJobs(context.Background(), ListParams{EmpresaID: "e1"})
	if err != nil {
		t.Fatalf("ListJobs: %v", err)
	}
	if len(jobs) != 2 || total != 2 {
		t.Errorf("got %d jobs, total %d", len(jobs), total)
	}
}

func TestService_GetJobProgress(t *testing.T) {
	repo := &mockRepo{
		progress: []*SyncJobProgress{{Executor: "clientes", Status: "concluido"}},
	}
	svc := NewService(repo, &mockDispatcher{}, zerolog.Nop())

	progress, err := svc.GetJobProgress(context.Background(), "j1")
	if err != nil {
		t.Fatalf("GetJobProgress: %v", err)
	}
	if len(progress) != 1 || progress[0].Executor != "clientes" {
		t.Error("progresso incorreto")
	}
}

func TestForcarSync_JobAtivoExistente_Retorna409(t *testing.T) {
	repo := &mockRepo{
		job: &SyncJob{ID: "job-ativo", Status: "rodando", Tipo: "manual"},
	}
	svc := NewService(repo, &mockDispatcher{}, zerolog.Nop())

	_, err := svc.ForcarSync(context.Background(), "g1", "e1", ForcarSyncRequest{Tipo: "manual"})
	if err == nil {
		t.Fatal("esperava erro 409")
	}

	ae, ok := apperror.IsAppError(err)
	if !ok || ae.Code != 409 {
		t.Errorf("esperava erro 409, got %v", err)
	}

	detail := ae.Detail.(map[string]any)
	jobAtivo := detail["job_ativo"].(map[string]any)
	if jobAtivo["id"] != "job-ativo" {
		t.Errorf("id incorreto no detail: %v", jobAtivo["id"])
	}
}

func TestForcarSync_JobAnteriorConcluido_Permite(t *testing.T) {
	repo := &mockRepo{
		job: &SyncJob{ID: "job-antigo", Status: "concluido", Tipo: "manual"},
	}
	svc := NewService(repo, &mockDispatcher{}, zerolog.Nop())

	job, err := svc.ForcarSync(context.Background(), "g1", "e1", ForcarSyncRequest{Tipo: "manual"})
	if err != nil {
		t.Fatalf("não deveria retornar erro: %v", err)
	}
	if job.ID != "job-1" { // mockRepo.InsertJob retorna job-1
		t.Errorf("job id: got %q want job-1", job.ID)
	}
}

func TestForcarSync_JobAnteriorErro_Permite(t *testing.T) {
	repo := &mockRepo{
		job: &SyncJob{ID: "job-antigo", Status: "erro", Tipo: "manual"},
	}
	svc := NewService(repo, &mockDispatcher{}, zerolog.Nop())

	_, err := svc.ForcarSync(context.Background(), "g1", "e1", ForcarSyncRequest{Tipo: "manual"})
	if err != nil {
		t.Fatalf("não deveria retornar erro: %v", err)
	}
}

func TestForcarSync_ExecutorSeletivoComJobGeralAtivo_409(t *testing.T) {
	repo := &mockRepo{
		job: &SyncJob{ID: "job-geral-ativo", Status: "rodando", Tipo: "manual"},
	}
	svc := NewService(repo, &mockDispatcher{}, zerolog.Nop())

	// Tenta iniciar um sync seletivo de 'clientes' enquanto o geral roda
	_, err := svc.ForcarSync(context.Background(), "g1", "e1", ForcarSyncRequest{Tipo: "manual", Executor: "clientes"})
	if err == nil {
		t.Fatal("esperava erro 409")
	}

	ae, ok := apperror.IsAppError(err)
	if !ok || ae.Code != 409 {
		t.Errorf("esperava erro 409, got %v", err)
	}
}

// Manutenção operacional — não exercitada por estes testes.
func (m *mockRepo) ConsultasAtivas(context.Context) ([]ConsultaAtiva, error)  { return nil, nil }
func (m *mockRepo) CancelarConsulta(context.Context, int32) (bool, error)     { return false, nil }
func (m *mockRepo) SchemaDoGrupo(context.Context, string) (string, error)     { return "", nil }
func (m *mockRepo) RefreshView(context.Context, string, string) (bool, error) { return false, nil }
