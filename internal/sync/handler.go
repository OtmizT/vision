package sync

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"

	"omie-sync-api/internal/apperror"
	"omie-sync-api/internal/auth"
	"omie-sync-api/internal/response"
)

type Handler struct {
	svc    Service
	jwtSvc auth.JWTService
	hub    *SSEHub
}

func NewHandler(svc Service, jwtSvc auth.JWTService, hub *SSEHub) *Handler {
	return &Handler{svc: svc, jwtSvc: jwtSvc, hub: hub}
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()

	// Rota SSE: aceita token via query param (?token=) pois EventSource não envia headers
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireAuthSSE(h.jwtSvc))
		r.Get("/{empresaID}/stream", h.Stream)
	})

	// Demais rotas: apenas Authorization: Bearer
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireAuth(h.jwtSvc))

		// Status e jobs: viewer com permissão também pode ver
		// Frescor dos dados do grupo. Um segmento só, então não colide com as
		// rotas /{empresaID}/... que exigem dois.
		r.Get("/ultima-atualizacao", h.UltimaAtualizacao)

		r.Get("/{empresaID}/status", h.GetStatus)
		r.Get("/{empresaID}/pages", h.GetPages)
		r.Get("/{empresaID}/jobs", h.ListJobs)
		r.Get("/{empresaID}/jobs/{jobID}/progress", h.GetJobProgress)

		// Forçar sync e configurar: apenas admin
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireRole("admin_global", "admin_grupo"))
			r.With(httprate.LimitByIP(5, 1*time.Minute)).Post("/{empresaID}/forcar", h.ForcarSync)
			r.Put("/{empresaID}/configurar", h.Configurar)

			// Configuração de executors por empresa
			r.Get("/{empresaID}/executors", h.GetExecutorConfigs)
			r.Put("/{empresaID}/executors/{executor}", h.UpdateExecutorConfig)
		})
	})

	return r
}

// GET /sync/{empresaID}/status
// GET /sync/ultima-atualizacao
//
// O grupo vem das claims, não da query: aceitar grupo por parâmetro abriria
// leitura do frescor de outro cliente.
func (h *Handler) UltimaAtualizacao(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok || claims.GrupoID == "" {
		response.OK(w, &UltimaAtualizacao{})
		return
	}

	res, err := h.svc.UltimaAtualizacao(r.Context(), claims.GrupoID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "erro ao buscar última atualização", err)
		return
	}

	response.OK(w, res)
}

func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	empresaID := chi.URLParam(r, "empresaID")

	status, err := h.svc.GetStatus(r.Context(), empresaID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "erro ao buscar status", err)
		return
	}

	response.OK(w, status)
}

// GET /sync/{empresaID}/jobs
func (h *Handler) ListJobs(w http.ResponseWriter, r *http.Request) {
	empresaID := chi.URLParam(r, "empresaID")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))

	jobs, total, err := h.svc.ListJobs(r.Context(), ListParams{
		EmpresaID: empresaID, Page: page, PerPage: perPage,
	})
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "erro ao listar jobs", err)
		return
	}

	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 50
	}

	response.OKPaginated(w, jobs, response.Meta{
		Page: page, PerPage: perPage, Total: int(total),
	})
}

// POST /sync/{empresaID}/forcar
func (h *Handler) ForcarSync(w http.ResponseWriter, r *http.Request) {
	empresaID := chi.URLParam(r, "empresaID")

	claims, _ := auth.ClaimsFromContext(r.Context())
	grupoID := ""
	if claims != nil {
		grupoID = claims.GrupoID
	}

	var req ForcarSyncRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	job, err := h.svc.ForcarSync(r.Context(), grupoID, empresaID, req)
	if err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro ao forçar sync", err)
		return
	}

	response.Created(w, job)
}

// PUT /sync/{empresaID}/configurar
func (h *Handler) Configurar(w http.ResponseWriter, r *http.Request) {
	empresaID := chi.URLParam(r, "empresaID")

	var req ConfigurarRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "body inválido", err)
		return
	}

	ctrl, err := h.svc.Configurar(r.Context(), empresaID, req)
	if err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro ao configurar sync", err)
		return
	}

	response.OK(w, ctrl)
}

// GET /sync/{empresaID}/jobs/{jobID}/progress
func (h *Handler) GetJobProgress(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")

	progress, err := h.svc.GetJobProgress(r.Context(), jobID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "erro ao buscar progresso do job", err)
		return
	}

	response.OK(w, progress)
}

// GET /sync/{empresaID}/executors
func (h *Handler) GetExecutorConfigs(w http.ResponseWriter, r *http.Request) {
	empresaID := chi.URLParam(r, "empresaID")

	configs, err := h.svc.GetExecutorConfigs(r.Context(), empresaID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "erro ao buscar configurações de módulos", err)
		return
	}

	response.OK(w, configs)
}

// PUT /sync/{empresaID}/executors/{executor}
func (h *Handler) UpdateExecutorConfig(w http.ResponseWriter, r *http.Request) {
	empresaID := chi.URLParam(r, "empresaID")
	executor := chi.URLParam(r, "executor")

	claims, _ := auth.ClaimsFromContext(r.Context())
	updatedBy := ""
	if claims != nil {
		updatedBy = claims.UserID
	}

	var req UpdateExecutorConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "body inválido", err)
		return
	}

	config, err := h.svc.UpdateExecutorConfig(r.Context(), empresaID, executor, req, updatedBy)
	if err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro ao atualizar configuração do módulo", err)
		return
	}

	response.OK(w, config)
}

// GET /sync/{empresaID}/stream
func (h *Handler) Stream(w http.ResponseWriter, r *http.Request) {
	empresaID := chi.URLParam(r, "empresaID")

	// Verifica suporte a flushing
	flusher, ok := w.(http.Flusher)
	if !ok {
		response.Error(w, http.StatusInternalServerError, "streaming não suportado", nil)
		return
	}

	// Headers SSE obrigatórios
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // desativa buffer do nginx
	w.WriteHeader(http.StatusOK)

	// Remove o WriteTimeout global (15s) apenas nesta conexão. Como o heartbeat
	// também é de 15s, o deadline vencia junto com ele e o stream caía o tempo
	// todo — no browser aparecia como ERR_HTTP2_PROTOCOL_ERROR. As demais rotas
	// mantêm o timeout do servidor.
	// Se falhar (ResponseWriter sem Unwrap na cadeia de middlewares), o stream ainda
	// funciona — apenas volta a cair a cada 15s, como antes desta correção.
	_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})

	ch, cancel := h.hub.Subscribe(empresaID)
	defer cancel()

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return // cliente desconectou
		case <-heartbeat.C:
			_, _ = fmt.Fprintf(w, "event: heartbeat\ndata: {}\n\n")
			flusher.Flush()
		case evt, ok := <-ch:
			if !ok {
				return
			}
			b, _ := json.Marshal(evt.Data)
			_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", evt.Type, b)
			flusher.Flush()
		}
	}
}

func (h *Handler) AdminOverview(w http.ResponseWriter, r *http.Request) {
	data, err := h.svc.GetAdminOverview(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "erro ao buscar overview", err)
		return
	}
	response.OK(w, data)
}

// AdminEmpresasSync: todas as empresas de todos os grupos, com o estado de
// sync. Alimenta a primeira aba do Sync Control.
func (h *Handler) AdminEmpresasSync(w http.ResponseWriter, r *http.Request) {
	rows, err := h.svc.ListEmpresasSyncGeral(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "erro ao listar empresas", err)
		return
	}
	response.OK(w, rows)
}

func (h *Handler) AdminJobsAtivos(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.svc.GetJobsAtivos(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "erro ao buscar jobs ativos", err)
		return
	}
	response.OK(w, jobs)
}

func (h *Handler) AdminCancelarJob(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")
	if err := h.svc.CancelarJob(r.Context(), jobID); err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro ao cancelar job", err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) AdminStartupRecovery(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.StartupRecovery(r.Context()); err != nil {
		response.Error(w, http.StatusInternalServerError, "erro no recovery", err)
		return
	}
	response.OK(w, map[string]string{"message": "recovery executado"})
}

func (h *Handler) AdminDLQ(w http.ResponseWriter, r *http.Request) {
	pages, err := h.svc.GetDLQPages(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "erro ao buscar DLQ", err)
		return
	}
	response.OK(w, pages)
}

func (h *Handler) AdminRetryPage(w http.ResponseWriter, r *http.Request) {
	pageID := chi.URLParam(r, "pageID")
	if err := h.svc.RetryDLQPage(r.Context(), pageID); err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro ao agendar retry", err)
		return
	}
	response.OK(w, map[string]string{"message": "página agendada para retry"})
}

func (h *Handler) GetPages(w http.ResponseWriter, r *http.Request) {
	empresaID := chi.URLParam(r, "empresaID")
	jobID := r.URL.Query().Get("job_id")
	pages, err := h.svc.GetPagesByEmpresa(r.Context(), empresaID, jobID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "erro ao buscar páginas", err)
		return
	}
	response.OK(w, pages)
}
