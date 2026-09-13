package ia_config

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"omie-sync-api/internal/apperror"
	"omie-sync-api/internal/auth"
	"omie-sync-api/internal/response"
)

type Handler struct {
	svc    Service
	jwtSvc auth.JWTService
}

func NewHandler(svc Service, jwtSvc auth.JWTService) *Handler {
	return &Handler{svc: svc, jwtSvc: jwtSvc}
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(auth.RequireAuth(h.jwtSvc))
	// Configuração de plataforma, como a da Omie: a credencial é da OTM e
	// ligar um grupo é decidir mandar os dados daquele cliente para fora.
	r.Use(auth.RequireRole("admin_global"))

	r.Get("/", h.Get)
	r.Put("/", h.Update)
	r.Get("/grupos", h.ListGrupos)
	r.Put("/grupos/{grupoID}", h.SetGrupo)

	return r
}

// GET /admin/ia-config
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.svc.Get(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "erro ao buscar configuração", err)
		return
	}
	response.OK(w, cfg)
}

// PUT /admin/ia-config
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
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

	cfg, err := h.svc.Update(r.Context(), req, claims.UserID)
	if err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro ao salvar configuração", err)
		return
	}
	response.OK(w, cfg)
}

// GET /admin/ia-config/grupos
func (h *Handler) ListGrupos(w http.ResponseWriter, r *http.Request) {
	grupos, err := h.svc.ListGrupos(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "erro ao listar grupos", err)
		return
	}
	response.OK(w, grupos)
}

// PUT /admin/ia-config/grupos/{grupoID}
func (h *Handler) SetGrupo(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "não autenticado")
		return
	}

	var req SetGrupoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "body inválido", err)
		return
	}

	if err := h.svc.SetGrupo(r.Context(), chi.URLParam(r, "grupoID"), req.Ativa, claims.UserID); err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro ao alterar grupo", err)
		return
	}
	response.NoContent(w)
}
