package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"omie-sync-api/sqlc/generated"
)

type Repository interface {
	InsertJob(ctx context.Context, empresaID, tipo, executor string) (*SyncJob, error)
	GetJobByID(ctx context.Context, id string) (*SyncJob, error)
	ListJobs(ctx context.Context, empresaID string, limit, offset int32) ([]*SyncJob, error)
	CountJobs(ctx context.Context, empresaID string) (int64, error)
	UpdateJobStatus(ctx context.Context, id, status, erro string, iniciadoAt, concluidoAt *time.Time) (*SyncJob, error)
	GetControl(ctx context.Context, empresaID string) (*SyncControl, error)
	// Frescor dos dados do grupo, para o indicador no cabeçalho.
	UltimoSyncDoGrupo(ctx context.Context, grupoID string) (*time.Time, error)

	// Manutenção operacional (admin_global)
	ConsultasAtivas(ctx context.Context) ([]ConsultaAtiva, error)
	CancelarConsulta(ctx context.Context, pid int32) (bool, error)
	SchemaDoGrupo(ctx context.Context, grupoID string) (string, error)
	RefreshView(ctx context.Context, schema, view string) (bool, error)
	UpsertControl(ctx context.Context, empresaID string, ativo bool, intervaloIncrementalMin, intervaloFullDias int, proximoSyncAt, proximoFullSyncAt *time.Time) (*SyncControl, error)
	UpdateControlAfterRun(ctx context.Context, empresaID, tipo string) error
	AdvanceScheduleOnDispatch(ctx context.Context, empresaID, tipo string) error
	GetJobProgress(ctx context.Context, jobID string) ([]*SyncJobProgress, error)

	MarkStaleJobs(ctx context.Context) (int64, error)
	UpdateJobHeartbeat(ctx context.Context, jobID string) error

	GetJobsOverview(ctx context.Context) ([]JobStatusCount, error)
	GetJobsAtivos(ctx context.Context) ([]JobAtivoRow, error)
	ListEmpresasSyncGeral(ctx context.Context) ([]EmpresaSyncRow, error)
	CancelarJob(ctx context.Context, jobID string) error

	InsertJobPage(ctx context.Context, jobID, modulo string, pagina, totalPaginas int) error
	GetPendingPages(ctx context.Context, jobID string, limit int) ([]JobPage, error)
	CountPendingPages(ctx context.Context, jobID string) (int64, error)

	ClaimPageForProcessing(ctx context.Context, pageID string) (*JobPage, error)
	MarkPageConcluido(ctx context.Context, pageID string, registros int) error
	MarkPageErro(ctx context.Context, pageID string, erro string, proximoRetry time.Time) error
	MarkPageCancelado(ctx context.Context, jobID string) error

	GetDLQPages(ctx context.Context) ([]DLQPageRow, error)
	RetryDLQPage(ctx context.Context, pageID string) error

	GetPagesByJob(ctx context.Context, jobID string) ([]PageRow, error)
	GetLatestJobIDByEmpresa(ctx context.Context, empresaID string) (string, error)

	// Configuração de executors por empresa
	GetExecutorConfigs(ctx context.Context, empresaID string) ([]*EmpresaExecutorConfig, error)
	UpsertExecutorConfig(ctx context.Context, empresaID, executor string, ativo bool, notas *string, updatedBy string) (*EmpresaExecutorConfig, error)
	GetEnabledExecutors(ctx context.Context, empresaID string) (map[string]bool, error)

	// Verificação de concorrência
	GetJobAtivo(ctx context.Context, empresaID string) (*JobAtivoResult, error)
}

type JobAtivoResult struct {
	Total      int
	ID         string
	Tipo       string
	Status     string
	IniciadoAt *time.Time
}

type repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &repository{pool: pool}
}

func (r *repository) InsertJob(ctx context.Context, empresaID, tipo, executor string) (*SyncJob, error) {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(empresaID); err != nil {
		return nil, fmt.Errorf("sync.repository.InsertJob scan uuid: %w", err)
	}
	var execPtr *string
	if executor != "" {
		execPtr = &executor
	}
	row, err := q.InsertSyncJob(ctx, sqlcgen.InsertSyncJobParams{EmpresaID: uid, Tipo: tipo, Executor: execPtr})
	if err != nil {
		return nil, fmt.Errorf("sync.repository.InsertJob: %w", err)
	}
	return toJob(row), nil
}

func (r *repository) GetJobByID(ctx context.Context, id string) (*SyncJob, error) {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(id); err != nil {
		return nil, fmt.Errorf("sync.repository.GetJobByID scan uuid: %w", err)
	}
	row, err := q.GetSyncJobByID(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("sync.repository.GetJobByID: %w", err)
	}
	return toJob(row), nil
}

func (r *repository) ListJobs(ctx context.Context, empresaID string, limit, offset int32) ([]*SyncJob, error) {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(empresaID); err != nil {
		return nil, fmt.Errorf("sync.repository.ListJobs scan uuid: %w", err)
	}
	rows, err := q.ListSyncJobsByEmpresa(ctx, sqlcgen.ListSyncJobsByEmpresaParams{EmpresaID: uid, Limit: limit, Offset: offset})
	if err != nil {
		return nil, fmt.Errorf("sync.repository.ListJobs: %w", err)
	}
	result := make([]*SyncJob, len(rows))
	for i, row := range rows {
		result[i] = toJob(row)
	}
	return result, nil
}

func (r *repository) CountJobs(ctx context.Context, empresaID string) (int64, error) {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(empresaID); err != nil {
		return 0, fmt.Errorf("sync.repository.CountJobs scan uuid: %w", err)
	}
	n, err := q.CountSyncJobsByEmpresa(ctx, uid)
	if err != nil {
		return 0, fmt.Errorf("sync.repository.CountJobs: %w", err)
	}
	return n, nil
}

func (r *repository) UpdateJobStatus(ctx context.Context, id, status, erro string, iniciadoAt, concluidoAt *time.Time) (*SyncJob, error) {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(id); err != nil {
		return nil, fmt.Errorf("sync.repository.UpdateJobStatus scan uuid: %w", err)
	}

	erroText := pgtype.Text{String: erro, Valid: erro != ""}
	var ini, conc pgtype.Timestamptz
	if iniciadoAt != nil {
		_ = ini.Scan(*iniciadoAt)
	}
	if concluidoAt != nil {
		_ = conc.Scan(*concluidoAt)
	}

	row, err := q.UpdateSyncJobStatus(ctx, sqlcgen.UpdateSyncJobStatusParams{
		ID:          uid,
		Status:      status,
		Erro:        erroText,
		IniciadoAt:  ini,
		ConcluidoAt: conc,
	})
	if err != nil {
		return nil, fmt.Errorf("sync.repository.UpdateJobStatus: %w", err)
	}
	return toJob(row), nil
}

func (r *repository) GetControl(ctx context.Context, empresaID string) (*SyncControl, error) {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(empresaID); err != nil {
		return nil, fmt.Errorf("sync.repository.GetControl scan uuid: %w", err)
	}
	row, err := q.GetSyncControl(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("sync.repository.GetControl: %w", err)
	}
	return toControlFromRow(row), nil
}

// UltimoSyncDoGrupo devolve nil quando nenhuma empresa do grupo sincronizou
// ainda — MAX() sobre conjunto vazio traz NULL, que não é erro.
func (r *repository) UltimoSyncDoGrupo(ctx context.Context, grupoID string) (*time.Time, error) {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(grupoID); err != nil {
		return nil, fmt.Errorf("sync.repository.UltimoSyncDoGrupo scan uuid: %w", err)
	}
	ts, err := q.GetUltimoSyncDoGrupo(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("sync.repository.UltimoSyncDoGrupo: %w", err)
	}
	if !ts.Valid {
		return nil, nil
	}
	t := ts.Time
	return &t, nil
}

func (r *repository) UpsertControl(ctx context.Context, empresaID string, ativo bool, intervaloIncrementalMin, intervaloFullDias int, proximoSyncAt, proximoFullSyncAt *time.Time) (*SyncControl, error) {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(empresaID); err != nil {
		return nil, fmt.Errorf("sync.repository.UpsertControl scan uuid: %w", err)
	}
	var prox, proxFull pgtype.Timestamptz
	if proximoSyncAt != nil {
		_ = prox.Scan(*proximoSyncAt)
	}
	if proximoFullSyncAt != nil {
		_ = proxFull.Scan(*proximoFullSyncAt)
	}
	row, err := q.UpsertSyncControl(ctx, sqlcgen.UpsertSyncControlParams{
		EmpresaID:               uid,
		Ativo:                   ativo,
		IntervaloIncrementalMin: int32(intervaloIncrementalMin),
		IntervaloFullDias:       int32(intervaloFullDias),
		ProximoSyncAt:           prox,
		ProximoFullSyncAt:       proxFull,
	})
	if err != nil {
		return nil, fmt.Errorf("sync.repository.UpsertControl: %w", err)
	}
	return toControl(row), nil
}

func (r *repository) UpdateControlAfterRun(ctx context.Context, empresaID, tipo string) error {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(empresaID); err != nil {
		return fmt.Errorf("sync.repository.UpdateControlAfterRun scan uuid: %w", err)
	}
	if err := q.UpdateSyncControlAfterRun(ctx, sqlcgen.UpdateSyncControlAfterRunParams{
		EmpresaID: uid,
		Tipo:      tipo,
	}); err != nil {
		return fmt.Errorf("sync.repository.UpdateControlAfterRun: %w", err)
	}
	return nil
}

func (r *repository) AdvanceScheduleOnDispatch(ctx context.Context, empresaID, tipo string) error {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(empresaID); err != nil {
		return fmt.Errorf("sync.repository.AdvanceScheduleOnDispatch scan uuid: %w", err)
	}
	if err := q.AdvanceSyncScheduleOnDispatch(ctx, sqlcgen.AdvanceSyncScheduleOnDispatchParams{
		EmpresaID: uid,
		Tipo:      tipo,
	}); err != nil {
		return fmt.Errorf("sync.repository.AdvanceScheduleOnDispatch: %w", err)
	}
	return nil
}

func (r *repository) GetJobProgress(ctx context.Context, jobID string) ([]*SyncJobProgress, error) {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(jobID); err != nil {
		return nil, fmt.Errorf("sync.repository.GetJobProgress scan uuid: %w", err)
	}
	rows, err := q.GetSyncJobProgress(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("sync.repository.GetJobProgress: %w", err)
	}
	result := make([]*SyncJobProgress, len(rows))
	for i, row := range rows {
		result[i] = toProgress(row)
	}
	return result, nil
}

func (r *repository) MarkStaleJobs(ctx context.Context) (int64, error) {
	q := sqlcgen.New(r.pool)
	n, err := q.MarkStaleJobs(ctx)
	if err != nil {
		return 0, fmt.Errorf("syncRepository.MarkStaleJobs: %w", err)
	}
	return n, nil
}

func (r *repository) UpdateJobHeartbeat(ctx context.Context, jobID string) error {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(jobID); err != nil {
		return fmt.Errorf("sync.repository.UpdateJobHeartbeat scan uuid: %w", err)
	}
	if err := q.UpdateJobHeartbeat(ctx, uid); err != nil {
		return fmt.Errorf("syncRepository.UpdateJobHeartbeat: %w", err)
	}
	return nil
}

func (r *repository) GetJobsOverview(ctx context.Context) ([]JobStatusCount, error) {
	q := sqlcgen.New(r.pool)
	rows, err := q.GetJobsOverview(ctx)
	if err != nil {
		return nil, fmt.Errorf("syncRepository.GetJobsOverview: %w", err)
	}
	result := make([]JobStatusCount, len(rows))
	for i, row := range rows {
		result[i] = JobStatusCount{
			Status: row.Status,
			Total:  row.Total,
		}
	}
	return result, nil
}

func (r *repository) GetJobsAtivos(ctx context.Context) ([]JobAtivoRow, error) {
	q := sqlcgen.New(r.pool)
	rows, err := q.GetJobsAtivos(ctx)
	if err != nil {
		return nil, fmt.Errorf("syncRepository.GetJobsAtivos: %w", err)
	}
	result := make([]JobAtivoRow, len(rows))
	for i, row := range rows {
		result[i] = JobAtivoRow{
			ID:          uuidToStr(row.ID),
			EmpresaID:   uuidToStr(row.EmpresaID),
			EmpresaNome: row.EmpresaNome,
			GrupoNome:   row.GrupoNome,
			Tipo:        row.Tipo,
			Status:      row.Status,
			IniciadoAt:  row.IniciadoAt.Time,
			IsZumbi:     row.IsZumbi,
		}
		if row.UltimoHeartbeatAt.Valid {
			t := row.UltimoHeartbeatAt.Time
			result[i].UltimoHeartbeatAt = &t
		}
	}
	return result, nil
}

/*
ListEmpresasSyncGeral devolve todas as empresas de todos os grupos com o estado
de sincronizacao.

O heartbeat nao vai para a struct de saida: ele so serve para decidir se o job
esta zumbi, e essa decisao e tomada aqui uma vez, em vez de repetir a regra dos
10 minutos na tela. Ver PrazoHeartbeat.
*/
func (r *repository) ListEmpresasSyncGeral(ctx context.Context) ([]EmpresaSyncRow, error) {
	q := sqlcgen.New(r.pool)
	rows, err := q.ListEmpresasSyncGeral(ctx)
	if err != nil {
		return nil, fmt.Errorf("syncRepository.ListEmpresasSyncGeral: %w", err)
	}

	agora := time.Now()
	out := make([]EmpresaSyncRow, len(rows))
	for i, row := range rows {
		e := EmpresaSyncRow{
			GrupoID:                 uuidToStr(row.GrupoID),
			GrupoNome:               row.GrupoNome,
			EmpresaID:               uuidToStr(row.EmpresaID),
			EmpresaNome:             row.EmpresaNome,
			Status:                  row.EmpresaStatus,
			StatusSync:              row.EmpresaStatusSync,
			SyncAtivo:               row.SyncAtivo,
			IntervaloIncrementalMin: row.IntervaloIncrementalMin,
			IntervaloFullDias:       row.IntervaloFullDias,
			JobID:                   uuidToStr(row.JobID),
			JobTipo:                 row.JobTipo,
			JobStatus:               row.JobStatus,
			JobRegistros:            row.JobRegistros,
		}

		e.UltimoSyncAt = tsPtr(row.UltimoSyncAt)
		e.ProximoSyncAt = tsPtr(row.ProximoSyncAt)
		e.UltimoFullSyncAt = tsPtr(row.UltimoFullSyncAt)
		e.ProximoFullSyncAt = tsPtr(row.ProximoFullSyncAt)
		e.JobIniciadoAt = tsPtr(row.JobIniciadoAt)
		e.JobConcluidoAt = tsPtr(row.JobConcluidoAt)

		if row.JobErro.Valid && row.JobErro.String != "" {
			msg := row.JobErro.String
			e.JobErro = &msg
		}

		e.JobZumbi = row.JobStatus == "rodando" &&
			row.JobHeartbeatAt.Valid &&
			agora.Sub(row.JobHeartbeatAt.Time) > PrazoHeartbeat

		out[i] = e
	}
	return out, nil
}

// tsPtr converte um timestamp anulavel do pgx em ponteiro, que e como o JSON
// distingue "sem valor" de "epoch". Repetido a mao em varios lugares deste
// arquivo antes desta funcao existir.
func tsPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}

func (r *repository) CancelarJob(ctx context.Context, jobID string) error {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(jobID); err != nil {
		return fmt.Errorf("sync.repository.CancelarJob scan uuid: %w", err)
	}
	if err := q.CancelarJob(ctx, uid); err != nil {
		return fmt.Errorf("syncRepository.CancelarJob: %w", err)
	}
	return nil
}

func (r *repository) InsertJobPage(ctx context.Context, jobID, modulo string, pagina, totalPaginas int) error {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(jobID); err != nil {
		return fmt.Errorf("sync.repository.InsertJobPage scan uuid: %w", err)
	}
	err := q.InsertJobPage(ctx, sqlcgen.InsertJobPageParams{
		JobID:        uid,
		Modulo:       modulo,
		Pagina:       int32(pagina),
		TotalPaginas: int32(totalPaginas),
	})
	if err != nil {
		return fmt.Errorf("syncRepository.InsertJobPage: %w", err)
	}
	return nil
}

func (r *repository) GetPendingPages(ctx context.Context, jobID string, limit int) ([]JobPage, error) {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(jobID); err != nil {
		return nil, fmt.Errorf("sync.repository.GetPendingPages scan uuid: %w", err)
	}
	rows, err := q.GetPendingPages(ctx, sqlcgen.GetPendingPagesParams{
		JobID:      uid,
		LimitCount: int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("syncRepository.GetPendingPages: %w", err)
	}
	result := make([]JobPage, len(rows))
	for i, row := range rows {
		result[i] = JobPage{
			ID:            uuidToStr(row.ID),
			JobID:         uuidToStr(row.JobID),
			Modulo:        row.Modulo,
			Pagina:        int(row.Pagina),
			TotalPaginas:  int(row.TotalPaginas),
			Tentativas:    int(row.Tentativas),
			MaxTentativas: int(row.MaxTentativas),
		}
		if row.ProximoRetryAt.Valid {
			t := row.ProximoRetryAt.Time
			result[i].ProximoRetryAt = &t
		}
	}
	return result, nil
}

func (r *repository) CountPendingPages(ctx context.Context, jobID string) (int64, error) {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(jobID); err != nil {
		return 0, fmt.Errorf("sync.repository.CountPendingPages scan uuid: %w", err)
	}
	n, err := q.CountPendingPages(ctx, uid)
	if err != nil {
		return 0, fmt.Errorf("syncRepository.CountPendingPages: %w", err)
	}
	return n, nil
}

func (r *repository) ClaimPageForProcessing(ctx context.Context, pageID string) (*JobPage, error) {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(pageID); err != nil {
		return nil, fmt.Errorf("sync.repository.ClaimPageForProcessing scan uuid: %w", err)
	}
	row, err := q.ClaimPageForProcessing(ctx, uid)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("syncRepository.ClaimPageForProcessing: %w", err)
	}
	return &JobPage{
		ID:            uuidToStr(row.ID),
		JobID:         uuidToStr(row.JobID),
		Modulo:        row.Modulo,
		Pagina:        int(row.Pagina),
		TotalPaginas:  int(row.TotalPaginas),
		Tentativas:    int(row.Tentativas),
		MaxTentativas: int(row.MaxTentativas),
	}, nil
}

func (r *repository) MarkPageConcluido(ctx context.Context, pageID string, registros int) error {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(pageID); err != nil {
		return fmt.Errorf("sync.repository.MarkPageConcluido scan uuid: %w", err)
	}
	err := q.MarkPageConcluido(ctx, sqlcgen.MarkPageConcluidoParams{
		PageID:            uid,
		RegistrosGravados: int32(registros),
	})
	if err != nil {
		return fmt.Errorf("syncRepository.MarkPageConcluido: %w", err)
	}
	return nil
}

func (r *repository) MarkPageErro(ctx context.Context, pageID string, erro string, proximoRetry time.Time) error {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(pageID); err != nil {
		return fmt.Errorf("sync.repository.MarkPageErro scan uuid: %w", err)
	}
	var prox pgtype.Timestamptz
	if !proximoRetry.IsZero() {
		_ = prox.Scan(proximoRetry)
	}
	err := q.MarkPageErro(ctx, sqlcgen.MarkPageErroParams{
		PageID:         uid,
		Erro:           pgtype.Text{String: erro, Valid: true},
		ProximoRetryAt: prox,
	})
	if err != nil {
		return fmt.Errorf("syncRepository.MarkPageErro: %w", err)
	}
	return nil
}

func (r *repository) MarkPageCancelado(ctx context.Context, jobID string) error {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(jobID); err != nil {
		return fmt.Errorf("sync.repository.MarkPageCancelado scan uuid: %w", err)
	}
	err := q.MarkPageCancelado(ctx, uid)
	if err != nil {
		return fmt.Errorf("syncRepository.MarkPageCancelado: %w", err)
	}
	return nil
}

func (r *repository) GetDLQPages(ctx context.Context) ([]DLQPageRow, error) {
	q := sqlcgen.New(r.pool)
	rows, err := q.GetDLQPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("syncRepository.GetDLQPages: %w", err)
	}
	result := make([]DLQPageRow, len(rows))
	for i, row := range rows {
		result[i] = DLQPageRow{
			ID:            uuidToStr(row.ID),
			JobID:         uuidToStr(row.JobID),
			EmpresaNome:   row.EmpresaNome,
			GrupoNome:     row.GrupoNome,
			Modulo:        row.Modulo,
			Pagina:        int(row.Pagina),
			TotalPaginas:  int(row.TotalPaginas),
			Tentativas:    int(row.Tentativas),
			MaxTentativas: int(row.MaxTentativas),
		}
		if row.Erro.Valid {
			result[i].Erro = &row.Erro.String
		}
		if row.ConcluidoAt.Valid {
			t := row.ConcluidoAt.Time
			result[i].ConcluidoAt = &t
		}
	}
	return result, nil
}

func (r *repository) RetryDLQPage(ctx context.Context, pageID string) error {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(pageID); err != nil {
		return fmt.Errorf("sync.repository.RetryDLQPage scan uuid: %w", err)
	}
	err := q.RetryDLQPage(ctx, uid)
	if err != nil {
		return fmt.Errorf("syncRepository.RetryDLQPage: %w", err)
	}
	return nil
}

func (r *repository) GetPagesByJob(ctx context.Context, jobID string) ([]PageRow, error) {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(jobID); err != nil {
		return nil, fmt.Errorf("sync.repository.GetPagesByJob scan uuid: %w", err)
	}
	rows, err := q.GetPagesByJob(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("syncRepository.GetPagesByJob: %w", err)
	}
	result := make([]PageRow, len(rows))
	for i, row := range rows {
		result[i] = PageRow{
			ID:                uuidToStr(row.ID),
			Modulo:            row.Modulo,
			Pagina:            int(row.Pagina),
			TotalPaginas:      int(row.TotalPaginas),
			Status:            row.Status,
			Tentativas:        int(row.Tentativas),
			MaxTentativas:     int(row.MaxTentativas),
			RegistrosGravados: int(row.RegistrosGravados),
		}
		if row.Erro.Valid {
			result[i].Erro = &row.Erro.String
		}
		if row.ProximoRetryAt.Valid {
			t := row.ProximoRetryAt.Time
			result[i].ProximoRetryAt = &t
		}
		if row.IniciadoAt.Valid {
			t := row.IniciadoAt.Time
			result[i].IniciadoAt = &t
		}
		if row.ConcluidoAt.Valid {
			t := row.ConcluidoAt.Time
			result[i].ConcluidoAt = &t
		}
	}
	return result, nil
}

func (r *repository) GetLatestJobIDByEmpresa(ctx context.Context, empresaID string) (string, error) {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(empresaID); err != nil {
		return "", fmt.Errorf("sync.repository.GetLatestJobIDByEmpresa scan uuid: %w", err)
	}
	id, err := q.GetLatestJobIDByEmpresa(ctx, uid)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return "", nil
		}
		return "", fmt.Errorf("syncRepository.GetLatestJobIDByEmpresa: %w", err)
	}
	return uuidToStr(id), nil
}

func (r *repository) GetExecutorConfigs(ctx context.Context, empresaID string) ([]*EmpresaExecutorConfig, error) {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(empresaID); err != nil {
		return nil, err
	}
	rows, err := q.GetExecutorConfigsByEmpresa(ctx, uid)
	if err != nil {
		return nil, err
	}
	result := make([]*EmpresaExecutorConfig, len(rows))
	for i, row := range rows {
		result[i] = &EmpresaExecutorConfig{
			Executor:  row.Executor,
			Ativo:     row.Ativo,
			UpdatedAt: row.UpdatedAt.Time,
		}
		if row.Notas.Valid {
			result[i].Notas = &row.Notas.String
		}
		if row.UpdatedBy.Valid {
			s := uuidToStr(row.UpdatedBy)
			result[i].UpdatedBy = &s
		}
	}
	return result, nil
}

func (r *repository) UpsertExecutorConfig(ctx context.Context, empresaID, executor string, ativo bool, notas *string, updatedBy string) (*EmpresaExecutorConfig, error) {
	q := sqlcgen.New(r.pool)
	var uid, userUID pgtype.UUID
	if err := uid.Scan(empresaID); err != nil {
		return nil, err
	}
	if err := userUID.Scan(updatedBy); err != nil {
		return nil, err
	}

	var nText pgtype.Text
	if notas != nil {
		nText = pgtype.Text{String: *notas, Valid: true}
	}

	row, err := q.UpsertExecutorConfig(ctx, sqlcgen.UpsertExecutorConfigParams{
		EmpresaID: uid,
		Executor:  executor,
		Ativo:     ativo,
		Notas:     nText,
		UpdatedBy: userUID,
	})
	if err != nil {
		return nil, err
	}

	res := &EmpresaExecutorConfig{
		Executor:  row.Executor,
		Ativo:     row.Ativo,
		UpdatedAt: row.UpdatedAt.Time,
	}
	if row.Notas.Valid {
		res.Notas = &row.Notas.String
	}
	if row.UpdatedBy.Valid {
		s := uuidToStr(row.UpdatedBy)
		res.UpdatedBy = &s
	}
	return res, nil
}

func (r *repository) GetEnabledExecutors(ctx context.Context, empresaID string) (map[string]bool, error) {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(empresaID); err != nil {
		return nil, err
	}

	// Por padrão, todos são ativos. A query busca os que estão explicitamente desativados.
	rows, err := q.GetEnabledExecutorsByEmpresa(ctx, uid)
	if err != nil {
		return nil, err
	}

	// Mapa de desativados
	disabled := make(map[string]bool)
	for _, row := range rows {
		disabled[row] = true
	}

	return disabled, nil
}

func (r *repository) GetJobAtivo(ctx context.Context, empresaID string) (*JobAtivoResult, error) {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(empresaID); err != nil {
		return nil, fmt.Errorf("sync.repository.GetJobAtivo scan uuid: %w", err)
	}

	row, err := q.GetJobAtivo(ctx, uid)
	if err != nil {
		// sqlcgen.GetJobAtivo usa QueryRow, se não encontrar nada retorna erro
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("sync.repository.GetJobAtivo: %w", err)
	}

	if row.Total == 0 {
		return nil, nil
	}

	res := &JobAtivoResult{
		Total:  int(row.Total),
		ID:     uuidToStr(row.ID),
		Tipo:   row.Tipo,
		Status: row.Status,
	}
	if row.IniciadoAt.Valid {
		t := row.IniciadoAt.Time
		res.IniciadoAt = &t
	}

	return res, nil
}

// --- helpers ---

func toJob(row sqlcgen.EtlSyncJob) *SyncJob {
	j := &SyncJob{
		ID:        uuidToStr(row.ID),
		EmpresaID: uuidToStr(row.EmpresaID),
		Tipo:      row.Tipo,
		Status:    row.Status,
		Erro:      row.Erro.String,
		CreatedAt: row.CreatedAt.Time,
	}
	if row.Executor != nil {
		j.Executor = *row.Executor
	}
	if row.IniciadoAt.Valid {
		t := row.IniciadoAt.Time
		j.IniciadoAt = &t
	}
	if row.ConcluidoAt.Valid {
		t := row.ConcluidoAt.Time
		j.ConcluidoAt = &t
	}
	return j
}

func toControl(row sqlcgen.EtlSyncControl) *SyncControl {
	c := &SyncControl{
		ID:                      uuidToStr(row.ID),
		EmpresaID:               uuidToStr(row.EmpresaID),
		Ativo:                   row.Ativo,
		IntervaloIncrementalMin: int(row.IntervaloIncrementalMin),
		IntervaloFullDias:       int(row.IntervaloFullDias),
		CreatedAt:               row.CreatedAt.Time,
		UpdatedAt:               row.UpdatedAt.Time,
	}
	if row.UltimoSyncAt.Valid {
		t := row.UltimoSyncAt.Time
		c.UltimoSyncAt = &t
	}
	if row.ProximoSyncAt.Valid {
		t := row.ProximoSyncAt.Time
		c.ProximoSyncAt = &t
	}
	if row.UltimoFullSyncAt.Valid {
		t := row.UltimoFullSyncAt.Time
		c.UltimoFullSyncAt = &t
	}
	if row.ProximoFullSyncAt.Valid {
		t := row.ProximoFullSyncAt.Time
		c.ProximoFullSyncAt = &t
	}
	return c
}

func toControlFromRow(row sqlcgen.GetSyncControlRow) *SyncControl {
	c := &SyncControl{
		ID:                      uuidToStr(row.ID),
		EmpresaID:               uuidToStr(row.EmpresaID),
		Ativo:                   row.Ativo,
		IntervaloIncrementalMin: int(row.IntervaloIncrementalMin),
		IntervaloFullDias:       int(row.IntervaloFullDias),
		CreatedAt:               row.CreatedAt.Time,
		UpdatedAt:               row.UpdatedAt.Time,
	}
	if row.UltimoSyncAt.Valid {
		t := row.UltimoSyncAt.Time
		c.UltimoSyncAt = &t
	}
	if row.ProximoSyncAt.Valid {
		t := row.ProximoSyncAt.Time
		c.ProximoSyncAt = &t
	}
	if row.UltimoFullSyncAt.Valid {
		t := row.UltimoFullSyncAt.Time
		c.UltimoFullSyncAt = &t
	}
	if row.ProximoFullSyncAt.Valid {
		t := row.ProximoFullSyncAt.Time
		c.ProximoFullSyncAt = &t
	}
	return c
}

func uuidToStr(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		u.Bytes[0], u.Bytes[1], u.Bytes[2], u.Bytes[3],
		u.Bytes[4], u.Bytes[5],
		u.Bytes[6], u.Bytes[7],
		u.Bytes[8], u.Bytes[9],
		u.Bytes[10], u.Bytes[11], u.Bytes[12], u.Bytes[13], u.Bytes[14], u.Bytes[15])
}

func toProgress(row sqlcgen.GetSyncJobProgressRow) *SyncJobProgress {
	p := &SyncJobProgress{
		Executor:      row.Executor,
		Status:        row.Status,
		RegistrosProc: int(row.RegistrosProc),
		UpdatedAt:     row.UpdatedAt.Time,
	}

	if row.PaginaAtual != nil {
		v := int(*row.PaginaAtual)
		p.PaginaAtual = &v
	}
	if row.TotalPaginas != nil {
		v := int(*row.TotalPaginas)
		p.TotalPaginas = &v
	}
	if row.RegistrosTotal != nil {
		v := int(*row.RegistrosTotal)
		p.RegistrosTotal = &v
	}
	if row.Erro.Valid {
		p.Erro = &row.Erro.String
	}
	if row.IniciadoAt.Valid {
		t := row.IniciadoAt.Time
		p.IniciadoAt = &t
	}
	if row.ConcluidoAt.Valid {
		t := row.ConcluidoAt.Time
		p.ConcluidoAt = &t
	}

	// Novos campos para inspeção de payload
	if len(row.UltimoPayload) > 0 {
		_ = json.Unmarshal(row.UltimoPayload, &p.UltimoPayload)
	}
	if len(row.UltimoResponse) > 0 {
		_ = json.Unmarshal(row.UltimoResponse, &p.UltimoResponse)
	}
	if len(row.ErroPayload) > 0 {
		_ = json.Unmarshal(row.ErroPayload, &p.ErroPayload)
	}
	if row.ErroResponse.Valid {
		p.ErroResponse = &row.ErroResponse.String
	}

	return p
}
