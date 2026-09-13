package ia

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"omie-sync-api/internal/apperror"
	"omie-sync-api/internal/auth"
	"omie-sync-api/internal/response"
)

type Handler struct {
	svc     Service
	jwtSvc  auth.JWTService
	membros auth.MembroChecker
	// limite é criado UMA vez no wire. Instanciar por request criaria um
	// limitador novo a cada chamada, e nada seria limitado.
	limite func(http.Handler) http.Handler
}

func NewHandler(svc Service, jwtSvc auth.JWTService, membros auth.MembroChecker, limite func(http.Handler) http.Handler) *Handler {
	return &Handler{svc: svc, jwtSvc: jwtSvc, membros: membros, limite: limite}
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(auth.RequireAuth(h.jwtSvc))
	// Sem RequireRole: admin de grupo e viewer usam o chat. Ele não revela nada
	// além do que a pessoa já vê no dashboard — as ferramentas rodam o mesmo
	// código, com o mesmo isolamento.
	r.Use(auth.RequireGrupoMembro(h.membros))

	r.Get("/disponivel", h.Disponivel)
	r.Get("/conversa", h.Historico)
	r.Delete("/conversa", h.Limpar)
	// Só a pergunta é limitada: é a única que custa dinheiro.
	r.With(h.limite).Post("/chat", h.Chat)

	return r
}

/*
grupoDoContexto: o grupo vem SEMPRE das claims.

Não há parâmetro de rota nem de corpo para escolher grupo. É a mesma garantia
das ferramentas, um nível acima.

Contexto de plataforma não tem grupo — e o assistente é sobre dados de um
cliente, então ali ele não existe.
*/
func grupoDoContexto(r *http.Request) (claims *auth.JWTClaims, grupoID string, ok bool) {
	c, existe := auth.ClaimsFromContext(r.Context())
	if !existe || c.GrupoID == "" {
		return nil, "", false
	}
	return c, c.GrupoID, true
}

// GET /ia/disponivel
func (h *Handler) Disponivel(w http.ResponseWriter, r *http.Request) {
	_, grupoID, ok := grupoDoContexto(r)
	if !ok {
		// Sem grupo não há assistente, mas também não é erro: a tela só não
		// mostra o botão.
		response.OK(w, map[string]bool{"disponivel": false})
		return
	}

	disponivel, err := h.svc.Disponivel(r.Context(), grupoID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "erro ao verificar disponibilidade", err)
		return
	}
	response.OK(w, map[string]bool{"disponivel": disponivel})
}

// POST /ia/chat
func (h *Handler) Chat(w http.ResponseWriter, r *http.Request) {
	claims, grupoID, ok := grupoDoContexto(r)
	if !ok {
		response.Forbidden(w, "o assistente só funciona dentro de um grupo")
		return
	}

	var req PerguntaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "body inválido", err)
		return
	}

	resp, err := h.svc.Perguntar(r.Context(), grupoID, claims.UserID, req)
	if err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		// A mensagem é genérica de propósito: o erro real pode carregar a URL
		// do provedor ou parte do prompt, e o prompt tem dado financeiro.
		response.Error(w, http.StatusInternalServerError, "o assistente não conseguiu responder agora", err)
		return
	}
	response.OK(w, resp)
}

// GET /ia/conversa
func (h *Handler) Historico(w http.ResponseWriter, r *http.Request) {
	claims, grupoID, ok := grupoDoContexto(r)
	if !ok {
		response.OK(w, []Mensagem{})
		return
	}

	msgs, err := h.svc.Historico(r.Context(), grupoID, claims.UserID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "erro ao carregar a conversa", err)
		return
	}
	response.OK(w, msgs)
}

// DELETE /ia/conversa
func (h *Handler) Limpar(w http.ResponseWriter, r *http.Request) {
	claims, grupoID, ok := grupoDoContexto(r)
	if !ok {
		response.NoContent(w)
		return
	}

	if err := h.svc.Limpar(r.Context(), grupoID, claims.UserID); err != nil {
		response.Error(w, http.StatusInternalServerError, "erro ao limpar a conversa", err)
		return
	}
	response.NoContent(w)
}
