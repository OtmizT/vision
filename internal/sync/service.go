package sync

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"omie-sync-api/internal/apperror"
	"omie-sync-api/internal/webhooks"
)

// JobProcessor Ã© implementado pelo worker â€” evita import circular.
type JobProcessor interface {
	ProcessJob(ctx context.Context, jobID string) error
}

// JobSubmitter submete um job para execução controlada pelo semáforo.
// Implementado pelo WorkerPool — evita import circular.
type JobSubmitter interface {
	Submit(ctx context.Context, jobID string, fn func())
}

type Service interface {
	GetStatus(ctx context.Context, empresaID string) (*StatusResponse, error)
	ListJobs(ctx context.Context, params ListParams) ([]*SyncJob, int64, error)
	ForcarSync(ctx context.Context, grupoID, empresaID string, req ForcarSyncRequest) (*SyncJob, error)
	Configurar(ctx context.Context, empresaID string, req ConfigurarRequest) (*SyncControl, error)
	GetJobProgress(ctx context.Context, jobID string) ([]*SyncJobProgress, error)
	UltimaAtualizacao(ctx context.Context, grupoID string) (*UltimaAtualizacao, error)

	StartupRecovery(ctx context.Context) error

	// Manutenção operacional (admin_global)
	ListarConsultasAtivas(ctx context.Context) ([]ConsultaAtiva, error)
	CancelarConsulta(ctx context.Context, pid int32) (bool, error)
	RefreshViewsGrupo(ctx context.Context, grupoID string) ([]RefreshViewResultado, error)
	GetAdminOverview(ctx context.Context) (map[string]int64, error)
	GetJobsAtivos(ctx context.Context) ([]JobAtivoRow, error)
	ListEmpresasSyncGeral(ctx context.Context) ([]EmpresaSyncRow, error)
	CancelarJob(ctx context.Context, jobID string) error

	GetDLQPages(ctx context.Context) ([]DLQPageRow, error)
	RetryDLQPage(ctx context.Context, pageID string) error

	GetPagesByEmpresa(ctx context.Context, empresaID string, jobID string) ([]PageRow, error)

	// Configuração de executors por empresa
	GetExecutorConfigs(ctx context.Context, empresaID string) ([]*EmpresaExecutorConfig, error)
	UpdateExecutorConfig(ctx context.Context, empresaID, executor string, req UpdateExecutorConfigRequest, updatedBy string) (*EmpresaExecutorConfig, error)
}

type service struct {
	repo       Repository
	dispatcher webhooks.Dispatcher
	processor  JobProcessor // nil até o worker ser registrado
	submitter  JobSubmitter // nil até o pool ser registrado
	log        zerolog.Logger
}

func NewService(repo Repository, dispatcher webhooks.Dispatcher, log zerolog.Logger) Service {
	return &service{
		repo:       repo,
		dispatcher: dispatcher,
		log:        log.With().Str("component", "sync_service").Logger(),
	}
}

// SetProcessor registra o worker apÃ³s construÃ§Ã£o â€” resolve dependÃªncia circular.
func SetProcessor(svc Service, p JobProcessor) {
	if s, ok := svc.(*service); ok {
		s.processor = p
	}
}

// SetSubmitter registra o WorkerPool apos construcao — garante que ForcarSync
// tambem passa pelo semaforo de concorrencia.
func SetSubmitter(svc Service, sub JobSubmitter) {
	if s, ok := svc.(*service); ok {
		s.submitter = sub
	}
}

func (s *service) GetStatus(ctx context.Context, empresaID string) (*StatusResponse, error) {
	control, err := s.repo.GetControl(ctx, empresaID)
	if err != nil {
		// Sem controle ainda â€” retorna status vazio
		control = nil
	}

	jobs, err := s.repo.ListJobs(ctx, empresaID, 1, 0)
	if err != nil {
		return nil, fmt.Errorf("sync.service.GetStatus listar jobs: %w", err)
	}

	resp := &StatusResponse{EmpresaID: empresaID, Control: control}
	if len(jobs) > 0 {
		resp.UltimoJob = jobs[0]
	}
	return resp, nil
}

func (s *service) ListJobs(ctx context.Context, params ListParams) ([]*SyncJob, int64, error) {
	if params.PerPage <= 0 {
		params.PerPage = 50
	}
	if params.Page <= 0 {
		params.Page = 1
	}

	offset := int32((params.Page - 1) * params.PerPage)
	jobs, err := s.repo.ListJobs(ctx, params.EmpresaID, int32(params.PerPage), offset)
	if err != nil {
		return nil, 0, fmt.Errorf("sync.service.ListJobs: %w", err)
	}

	total, err := s.repo.CountJobs(ctx, params.EmpresaID)
	if err != nil {
		return nil, 0, fmt.Errorf("sync.service.ListJobs count: %w", err)
	}

	return jobs, total, nil
}

var validExecutors = map[string]bool{
	"clientes":               true,
	"categorias":             true,
	"departamentos":          true,
	"contas_correntes":       true,
	"contas_pagar":           true,
	"contas_receber":         true,
	"movimentos_financeiros": true,
	"extrato":                true,
	"ordens_servico":         true,
	"projetos":               true,
}

func (s *service) ForcarSync(ctx context.Context, grupoID, empresaID string, req ForcarSyncRequest) (*SyncJob, error) {
	// 1. Verificação de concorrência
	ativo, err := s.repo.GetJobAtivo(ctx, empresaID)
	if err != nil {
		return nil, fmt.Errorf("sync.service.ForcarSync verificar job ativo: %w", err)
	}
	if ativo != nil && ativo.Total > 0 {
		return nil, apperror.ConflictWithDetail("já existe um sync em andamento para esta empresa", map[string]any{
			"job_ativo": map[string]any{
				"id":          ativo.ID,
				"tipo":        ativo.Tipo,
				"status":      ativo.Status,
				"iniciado_at": ativo.IniciadoAt,
			},
		})
	}

	tipo := req.Tipo
	if tipo == "" {
		tipo = "manual"
	}

	if req.Executor != "" {
		if !validExecutors[req.Executor] {
			return nil, apperror.Unprocessable(fmt.Sprintf("executor inválido: %s", req.Executor))
		}

		// Verifica se o executor está desabilitado para a empresa
		disabled, err := s.repo.GetEnabledExecutors(ctx, empresaID)
		if err != nil {
			return nil, fmt.Errorf("sync.service.ForcarSync verificar habilitados: %w", err)
		}
		if disabled[req.Executor] {
			return nil, apperror.Unprocessable(fmt.Sprintf("executor '%s' está desabilitado para esta empresa", req.Executor))
		}
	}

	job, err := s.repo.InsertJob(ctx, empresaID, tipo, req.Executor)
	if err != nil {
		return nil, fmt.Errorf("sync.service.ForcarSync criar job: %w", err)
	}

	// Executa o job imediatamente — passa pelo semaforo se disponivel, senao goroutine direta
	if s.processor != nil {
		jobID := job.ID
		fn := func() {
			jobCtx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
			defer cancel()
			if err := s.processor.ProcessJob(jobCtx, jobID); err != nil {
				_ = err
			}
		}
		if s.submitter != nil {
			s.submitter.Submit(context.Background(), jobID, fn)
		} else {
			go fn()
		}
	}

	return job, nil
}

func (s *service) Configurar(ctx context.Context, empresaID string, req ConfigurarRequest) (*SyncControl, error) {
	// Validações de valores permitidos
	validIncremental := []int{60, 120, 240, 720}
	validFull := []int{5, 7, 15}

	incrementalOk := false
	for _, v := range validIncremental {
		if req.IntervaloIncrementalMin == v {
			incrementalOk = true
			break
		}
	}
	if !incrementalOk {
		return nil, apperror.Unprocessable("intervalo_incremental_min deve ser um de [60, 120, 240, 720]")
	}

	fullOk := false
	for _, v := range validFull {
		if req.IntervaloFullDias == v {
			fullOk = true
			break
		}
	}
	if !fullOk {
		return nil, apperror.Unprocessable("intervalo_full_dias deve ser um de [5, 7, 15]")
	}

	proximo := time.Now().Add(time.Duration(req.IntervaloIncrementalMin) * time.Minute)
	proximoFull := time.Now().Add(time.Duration(req.IntervaloFullDias) * 24 * time.Hour)

	control, err := s.repo.UpsertControl(ctx, empresaID, req.Ativo, req.IntervaloIncrementalMin, req.IntervaloFullDias, &proximo, &proximoFull)
	if err != nil {
		return nil, fmt.Errorf("sync.service.Configurar: %w", err)
	}

	return control, nil
}

func (s *service) GetJobProgress(ctx context.Context, jobID string) ([]*SyncJobProgress, error) {
	progress, err := s.repo.GetJobProgress(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("sync.service.GetJobProgress: %w", err)
	}
	return progress, nil
}

// UltimaAtualizacao informa quando os dados do grupo foram sincronizados por
// último. Serve ao indicador de frescor no cabeçalho.
//
// O REFRESH das views gerenciais roda imediatamente após cada sync, então este
// horário é um retrato fiel de quando o dashboard mudou. A exceção é o refresh
// manual pela rota de manutenção, que não toca em ultimo_sync_at.
func (s *service) UltimaAtualizacao(ctx context.Context, grupoID string) (*UltimaAtualizacao, error) {
	t, err := s.repo.UltimoSyncDoGrupo(ctx, grupoID)
	if err != nil {
		return nil, fmt.Errorf("syncService.UltimaAtualizacao: %w", err)
	}
	return &UltimaAtualizacao{UltimoSyncAt: t}, nil
}

func (s *service) StartupRecovery(ctx context.Context) error {
	n, err := s.repo.MarkStaleJobs(ctx)
	if err != nil {
		return fmt.Errorf("syncService.StartupRecovery: %w", err)
	}
	if n > 0 {
		s.log.Warn().Int64("jobs_recuperados", n).Msg("startup recovery: jobs zumbis marcados como erro")
	}
	return nil
}

func (s *service) GetAdminOverview(ctx context.Context) (map[string]int64, error) {
	counts, err := s.repo.GetJobsOverview(ctx)
	if err != nil {
		return nil, fmt.Errorf("syncService.GetAdminOverview: %w", err)
	}

	result := make(map[string]int64)
	for _, c := range counts {
		result[c.Status] = c.Total
	}
	return result, nil
}

// ListEmpresasSyncGeral: visao do admin global, todas as empresas de todos os
// grupos. Sem filtro de grupo de proposito — e a unica tela que precisa ver o
// parque inteiro.
func (s *service) ListEmpresasSyncGeral(ctx context.Context) ([]EmpresaSyncRow, error) {
	rows, err := s.repo.ListEmpresasSyncGeral(ctx)
	if err != nil {
		return nil, fmt.Errorf("syncService.ListEmpresasSyncGeral: %w", err)
	}
	return rows, nil
}

func (s *service) GetJobsAtivos(ctx context.Context) ([]JobAtivoRow, error) {
	jobs, err := s.repo.GetJobsAtivos(ctx)
	if err != nil {
		return nil, fmt.Errorf("syncService.GetJobsAtivos: %w", err)
	}
	return jobs, nil
}

func (s *service) CancelarJob(ctx context.Context, jobID string) error {
	if err := s.repo.CancelarJob(ctx, jobID); err != nil {
		return fmt.Errorf("syncService.CancelarJob: %w", err)
	}
	return nil
}

func (s *service) GetDLQPages(ctx context.Context) ([]DLQPageRow, error) {
	pages, err := s.repo.GetDLQPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("syncService.GetDLQPages: %w", err)
	}
	return pages, nil
}

func (s *service) RetryDLQPage(ctx context.Context, pageID string) error {
	if err := s.repo.RetryDLQPage(ctx, pageID); err != nil {
		return fmt.Errorf("syncService.RetryDLQPage: %w", err)
	}
	return nil
}

func (s *service) GetPagesByEmpresa(ctx context.Context, empresaID, jobID string) ([]PageRow, error) {
	targetJobID := jobID
	if targetJobID == "" {
		id, err := s.repo.GetLatestJobIDByEmpresa(ctx, empresaID)
		if err != nil {
			return nil, fmt.Errorf("syncService.GetPagesByEmpresa: %w", err)
		}
		if id == "" {
			return []PageRow{}, nil
		}
		targetJobID = id
	}
	pages, err := s.repo.GetPagesByJob(ctx, targetJobID)
	if err != nil {
		return nil, fmt.Errorf("syncService.GetPagesByEmpresa: %w", err)
	}
	return pages, nil
}

func (s *service) GetExecutorConfigs(ctx context.Context, empresaID string) ([]*EmpresaExecutorConfig, error) {
	configs, err := s.repo.GetExecutorConfigs(ctx, empresaID)
	if err != nil {
		return nil, fmt.Errorf("sync.service.GetExecutorConfigs: %w", err)
	}

	// Mescla com a lista completa de executors (ausência = ativo)
	result := make([]*EmpresaExecutorConfig, 0, len(validExecutors))
	configMap := make(map[string]*EmpresaExecutorConfig)
	for _, c := range configs {
		configMap[c.Executor] = c
	}

	// Ordem fixa dos executors para o frontend
	order := []string{
		"categorias", "departamentos", "contas_correntes", "clientes",
		"contas_pagar", "contas_receber", "movimentos_financeiros",
		"extrato", "ordens_servico", "projetos",
	}

	for _, name := range order {
		if c, ok := configMap[name]; ok {
			result = append(result, c)
		} else {
			result = append(result, &EmpresaExecutorConfig{
				Executor: name,
				Ativo:    true,
			})
		}
	}

	return result, nil
}

func (s *service) UpdateExecutorConfig(ctx context.Context, empresaID, executor string, req UpdateExecutorConfigRequest, updatedBy string) (*EmpresaExecutorConfig, error) {
	if !validExecutors[executor] {
		return nil, apperror.Unprocessable(fmt.Sprintf("executor inválido: %s", executor))
	}

	// Regra: Não permitir desabilitar executor se houver job rodando
	if !req.Ativo {
		jobs, err := s.repo.ListJobs(ctx, empresaID, 5, 0)
		if err == nil {
			for _, j := range jobs {
				if j.Status == "rodando" || j.Status == "pendente" {
					return nil, apperror.Conflict("não é possível desativar módulos enquanto um job de sincronização está em andamento")
				}
			}
		}
	}

	config, err := s.repo.UpsertExecutorConfig(ctx, empresaID, executor, req.Ativo, req.Notas, updatedBy)
	if err != nil {
		return nil, fmt.Errorf("sync.service.UpdateExecutorConfig: %w", err)
	}

	return config, nil
}
