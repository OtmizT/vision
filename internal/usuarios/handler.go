package usuarios

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
	// Faltava aqui: a rota é /admin/grupos/{grupoID}/usuarios e só verificava o
	// papel. Um admin do cliente A trocava o UUID da URL e listava, criava,
	// editava e apagava usuários do cliente B.
	r.Use(auth.RequireGrupoMembro(h.membros))

	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{usuarioID}", h.GetByID)
	r.Put("/{usuarioID}", h.Update)
	r.Put("/{usuarioID}/password", h.UpdatePassword)
	r.Delete("/{usuarioID}", h.Delete)

	return r
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	grupoID := chi.URLParam(r, "grupoID")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))

	us, total, err := h.svc.List(r.Context(), ListParams{GrupoID: grupoID, Page: page, PerPage: perPage})
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "erro ao listar usuários", err)
		return
	}
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 50
	}
	response.OKPaginated(w, us, response.Meta{Page: page, PerPage: perPage, Total: int(total)})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	grupoID := chi.URLParam(r, "grupoID")

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "body inválido", err)
		return
	}

	result, err := h.svc.Create(r.Context(), grupoID, req)
	if err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro ao criar usuário", err)
		return
	}
	response.Created(w, result)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "usuarioID")
	u, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro ao buscar usuário", err)
		return
	}
	response.OK(w, u)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "usuarioID")

	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "não autenticado")
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "body inválido", err)
		return
	}

	u, err := h.svc.Update(r.Context(), id, claims.GrupoID, req)
	if err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro ao atualizar usuário", err)
		return
	}
	response.OK(w, u)
}

func (h *Handler) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "usuarioID")

	var req UpdatePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "body inválido", err)
		return
	}

	if err := h.svc.UpdatePassword(r.Context(), id, req); err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro ao atualizar senha", err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "usuarioID")

	if err := h.svc.Delete(r.Context(), id); err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro ao deletar usuário", err)
		return
	}
	response.NoContent(w)
}
