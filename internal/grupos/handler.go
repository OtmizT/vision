package grupos

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

	r.Get("/", h.List)

	// Criar grupo é ato de plataforma. Estava fora de qualquer bloco de papel:
	// qualquer autenticado — viewer inclusive — provisionava um schema novo.
	r.With(auth.RequireRole("admin_global")).Post("/", h.Create)

	r.Group(func(r chi.Router) {
		r.Use(auth.RequireRole("admin_global", "admin_grupo"))
		r.Use(auth.RequireGrupoMembro(h.membros))
		r.Get("/{grupoID}", h.GetByID)
		r.Put("/{grupoID}", h.Update)
		r.Delete("/{grupoID}", h.Delete)
	})

	return r
}

// GET /admin/grupos
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))

	grupos, total, err := h.svc.List(r.Context(), ListParams{Page: page, PerPage: perPage})
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "erro ao listar grupos", err)
		return
	}

	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 50
	}

	response.OKPaginated(w, grupos, response.Meta{
		Page:    page,
		PerPage: perPage,
		Total:   int(total),
	})
}

// POST /admin/grupos
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "body inválido", err)
		return
	}

	grupo, err := h.svc.Create(r.Context(), req)
	if err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro ao criar grupo", err)
		return
	}

	response.Created(w, grupo)
}

// GET /admin/grupos/{grupoID}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "grupoID")

	grupo, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro ao buscar grupo", err)
		return
	}

	response.OK(w, grupo)
}

// PUT /admin/grupos/{grupoID}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "grupoID")

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "body inválido", err)
		return
	}

	grupo, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro ao atualizar grupo", err)
		return
	}

	response.OK(w, grupo)
}

// DELETE /admin/grupos/{grupoID}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "grupoID")

	err := h.svc.Delete(r.Context(), id)
	if err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro ao deletar grupo", err)
		return
	}

	response.NoContent(w)
}
