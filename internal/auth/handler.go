package auth

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"

	"omie-sync-api/internal/apperror"
	"omie-sync-api/internal/response"
)

type Handler struct {
	svc    Service
	jwtSvc JWTService
}

func NewHandler(svc Service, jwtSvc JWTService) *Handler {
	return &Handler{svc: svc, jwtSvc: jwtSvc}
}

func (h *Handler) JWTService() JWTService {
	return h.jwtSvc
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()

	r.With(httprate.LimitByIP(10, 1*time.Minute)).Post("/login", h.Login)
	r.With(httprate.LimitByIP(10, 1*time.Minute)).Post("/select-grupo", h.SelectGrupo)
	r.Post("/logout", h.Logout)
	r.With(httprate.LimitByIP(20, 1*time.Minute)).Post("/refresh", h.Refresh)
	r.With(RequireAuth(h.jwtSvc)).Get("/me", h.Me)
	r.With(RequireAuth(h.jwtSvc)).Get("/grupos", h.Grupos)
	r.With(RequireAuth(h.jwtSvc)).Get("/contextos", h.Contextos)
	// Troca da propria senha, exigindo a atual. A tela de Perfil chamava o
	// endpoint administrativo, e por isso um viewer recebia 403 ao tentar
	// trocar a propria senha.
	r.With(RequireAuth(h.jwtSvc), httprate.LimitByIP(10, 1*time.Minute)).Put("/senha", h.TrocarSenha)
	r.With(RequireAuth(h.jwtSvc)).Post("/troca-grupo", h.TrocaGrupo)

	return r
}

// POST /auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "body inválido", err)
		return
	}
	if req.Email == "" || req.Password == "" {
		response.Error(w, http.StatusUnprocessableEntity, "email e password são obrigatórios", nil)
		return
	}

	resp, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro interno", err)
		return
	}

	response.OK(w, resp)
}

// POST /auth/logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		response.Error(w, http.StatusUnprocessableEntity, "refresh_token é obrigatório", nil)
		return
	}

	if err := h.svc.Logout(r.Context(), req.RefreshToken); err != nil {
		response.Error(w, http.StatusInternalServerError, "erro ao revogar token", err)
		return
	}

	response.NoContent(w)
}

// POST /auth/refresh
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		response.Error(w, http.StatusUnprocessableEntity, "refresh_token é obrigatório", nil)
		return
	}

	resp, err := h.svc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro ao renovar token", err)
		return
	}

	response.OK(w, resp)
}

// GET /auth/grupos
func (h *Handler) Grupos(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "não autenticado")
		return
	}

	grupos, err := h.svc.GetGrupos(r.Context(), claims.UserID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "erro ao buscar grupos", err)
		return
	}

	response.OK(w, grupos)
}

// POST /auth/select-grupo
func (h *Handler) SelectGrupo(w http.ResponseWriter, r *http.Request) {
	var req SelectGrupoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "body inválido", err)
		return
	}
	if req.PreAuthToken == "" {
		response.Error(w, http.StatusUnprocessableEntity, "pre_auth_token é obrigatório", nil)
		return
	}
	contexto := contextoOuGrupo(req.Contexto)
	if contexto == ContextoGrupo && req.GrupoID == "" {
		response.Error(w, http.StatusUnprocessableEntity, "grupo_id é obrigatório no contexto de grupo", nil)
		return
	}

	resp, err := h.svc.SelectGrupo(r.Context(), req.PreAuthToken, contexto, req.GrupoID)
	if err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro ao selecionar grupo", err)
		return
	}

	response.OK(w, resp)
}

// POST /auth/troca-grupo
func (h *Handler) TrocaGrupo(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "não autenticado")
		return
	}

	var req TrocaGrupoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "body inválido", err)
		return
	}
	contexto := contextoOuGrupo(req.Contexto)
	if contexto == ContextoGrupo && req.GrupoID == "" {
		response.Error(w, http.StatusUnprocessableEntity, "grupo_id é obrigatório no contexto de grupo", nil)
		return
	}

	resp, err := h.svc.TrocaGrupo(r.Context(), claims.UserID, contexto, req.GrupoID)
	if err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro ao trocar grupo", err)
		return
	}

	response.OK(w, resp)
}

/*
contextoOuGrupo lê um contexto vindo do cliente.

Vazio vira "grupo", e só isso: é o que um cliente antigo poderia ter querido
dizer, e nunca concede a plataforma por omissão. Valor desconhecido também cai
em grupo — quem decide se pode mesmo entrar é RoleEfetiva, no service; aqui só
se normaliza a entrada.
*/
func contextoOuGrupo(c Contexto) Contexto {
	if c == ContextoPlataforma {
		return ContextoPlataforma
	}
	return ContextoGrupo
}

// GET /auth/contextos
func (h *Handler) Contextos(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "não autenticado")
		return
	}

	ctxs, err := h.svc.GetContextos(r.Context(), claims.UserID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "erro ao buscar contextos", err)
		return
	}
	response.OK(w, ctxs)
}

// PUT /auth/senha
func (h *Handler) TrocarSenha(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "não autenticado")
		return
	}

	var req TrocaSenhaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "body inválido", err)
		return
	}
	if req.SenhaAtual == "" || req.SenhaNova == "" {
		response.Error(w, http.StatusUnprocessableEntity, "senha_atual e senha_nova são obrigatórias", nil)
		return
	}

	resp, err := h.svc.TrocarSenhaPropria(r.Context(), claims.UserID, claims.Contexto, claims.GrupoID, req)
	if err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro ao trocar senha", err)
		return
	}

	response.OK(w, resp)
}

// GET /auth/me
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "não autenticado")
		return
	}

	me, err := h.svc.Me(r.Context(), claims.UserID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "erro ao buscar usuário", err)
		return
	}

	// Role, GrupoID e Contexto vêm do JWT — refletem onde a pessoa está agora,
	// não o valor estático do banco (que pode diferir após troca de contexto).
	me.Role = claims.Role
	me.GrupoID = claims.GrupoID
	me.Contexto = claims.Contexto

	response.OK(w, me)
}
