package empresas

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"omie-sync-api/internal/apperror"
	"omie-sync-api/internal/auth"
	"omie-sync-api/internal/response"
)

type Handler struct {
	svc    Service
	jwtSvc auth.JWTService
	// membros verifica o vínculo usuário×grupo a cada request. Pode ser nil em
	// testes — ver auth.RequireGrupoMembro.
	membros auth.MembroChecker
}

func NewHandler(svc Service, jwtSvc auth.JWTService, membros auth.MembroChecker) *Handler {
	return &Handler{svc: svc, jwtSvc: jwtSvc, membros: membros}
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(auth.RequireAuth(h.jwtSvc))
	r.Use(auth.RequireRole("admin_global", "admin_grupo"))
	r.Use(auth.RequireGrupoMembro(h.membros))

	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{empresaID}", h.GetByID)
	r.Put("/{empresaID}", h.Update)
	r.Delete("/{empresaID}", h.Delete)
	r.Post("/{empresaID}/reativar", h.Reativar)

	return r
}

// GET /admin/grupos/{grupoID}/empresas
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	grupoID := chi.URLParam(r, "grupoID")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))

	empresas, total, err := h.svc.List(r.Context(), ListParams{
		GrupoID: grupoID, Page: page, PerPage: perPage,
	})
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "erro ao listar empresas", err)
		return
	}

	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 50
	}

	response.OKPaginated(w, empresas, response.Meta{
		Page: page, PerPage: perPage, Total: int(total),
	})
}

// POST /admin/grupos/{grupoID}/empresas
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	grupoID := chi.URLParam(r, "grupoID")

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "body inválido", err)
		return
	}

	empresa, err := h.svc.Create(r.Context(), grupoID, req)
	if err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro ao criar empresa", err)
		return
	}

	response.Created(w, empresa)
}

// GET /admin/grupos/{grupoID}/empresas/{empresaID}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	grupoID := chi.URLParam(r, "grupoID")
	id := chi.URLParam(r, "empresaID")

	empresa, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro ao buscar empresa", err)
		return
	}

	// Defesa em profundidade: garante que a empresa pertence ao grupo do path (previne IDOR)
	if empresa.GrupoID != grupoID {
		response.FromAppError(w, apperror.Forbidden("acesso negado"))
		return
	}

	response.OK(w, empresa)
}

// PUT /admin/grupos/{grupoID}/empresas/{empresaID}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	grupoID := chi.URLParam(r, "grupoID")
	id := chi.URLParam(r, "empresaID")

	// Verifica posse antes de decodificar o body (falha rápida)
	existing, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro ao buscar empresa", err)
		return
	}
	if existing.GrupoID != grupoID {
		response.FromAppError(w, apperror.Forbidden("acesso negado"))
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "body inválido", err)
		return
	}

	empresa, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro ao atualizar empresa", err)
		return
	}

	response.OK(w, empresa)
}

// DELETE /admin/grupos/{grupoID}/empresas/{empresaID}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	grupoID := chi.URLParam(r, "grupoID")
	id := chi.URLParam(r, "empresaID")

	existing, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro ao buscar empresa", err)
		return
	}
	if existing.GrupoID != grupoID {
		response.FromAppError(w, apperror.Forbidden("acesso negado"))
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro ao deletar empresa", err)
		return
	}

	response.NoContent(w)
}

// POST /admin/grupos/{grupoID}/empresas/{empresaID}/reativar
func (h *Handler) Reativar(w http.ResponseWriter, r *http.Request) {
	grupoID := chi.URLParam(r, "grupoID")
	id := chi.URLParam(r, "empresaID")

	existing, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro ao buscar empresa", err)
		return
	}
	if existing.GrupoID != grupoID {
		response.FromAppError(w, apperror.Forbidden("acesso negado"))
		return
	}

	if err := h.svc.Reativar(r.Context(), id); err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro ao reativar empresa", err)
		return
	}

	response.NoContent(w)
}
