package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"omie-sync-api/internal/audit"
	"omie-sync-api/internal/response"
)

type ctxKey string

const CtxKeyUserClaims ctxKey = "user_claims"

/*
rotasComSenhaProvisoria: o que ainda funciona enquanto a senha e provisoria.

Uma senha definida por administrador precisa ser trocada ANTES de qualquer outra
coisa — nao "assim que der". Obrigar isso so na tela seria teatro: bastaria
chamar a API direto para seguir usando a conta com a senha que outra pessoa
conhece.

A lista e minima e existe por um motivo cada:

	/auth/senha    o proprio caminho da troca;
	/auth/me       a tela precisa saber quem e para montar a tela de troca;
	/auth/logout   desistir e sair nao pode ficar bloqueado;
	/auth/refresh  a troca pode demorar mais que os 15 minutos do token.
*/
var rotasComSenhaProvisoria = map[string]bool{
	"/auth/senha":   true,
	"/auth/me":      true,
	"/auth/logout":  true,
	"/auth/refresh": true,
}

func bloqueadoPorSenhaProvisoria(claims *JWTClaims, path string) bool {
	return claims.SenhaProvisoria && !rotasComSenhaProvisoria[path]
}

// RequireAuth valida o Bearer token e injeta as claims no contexto.
// Aceita apenas o header Authorization: Bearer <token>.
func RequireAuth(jwtSvc JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearer(r)
			if token == "" {
				response.Unauthorized(w, "token não fornecido")
				return
			}

			claims, err := jwtSvc.Validate(token)
			if err != nil {
				response.Unauthorized(w, "token inválido ou expirado")
				return
			}

			// Registra quem é, para a auditoria. O middleware de auditoria roda
			// antes deste e não tem como saber — ver audit/ator.go.
			audit.AtorFromContext(r.Context()).Registrar(claims.UserID, claims.Email, claims.Role)

			if bloqueadoPorSenhaProvisoria(claims, r.URL.Path) {
				response.Forbidden(w, "troque a senha provisória antes de continuar")
				return
			}

			ctx := context.WithValue(r.Context(), CtxKeyUserClaims, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuthSSE valida o Bearer token aceitando também o query param ?token=.
// Deve ser usado APENAS nas rotas SSE (Server-Sent Events), onde o browser
// não permite enviar headers customizados via EventSource.
func RequireAuthSSE(jwtSvc JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearer(r)
			if token == "" {
				token = r.URL.Query().Get("token")
			}
			if token == "" {
				response.Unauthorized(w, "token não fornecido")
				return
			}

			claims, err := jwtSvc.Validate(token)
			if err != nil {
				response.Unauthorized(w, "token inválido ou expirado")
				return
			}

			// Registra quem é, para a auditoria. O middleware de auditoria roda
			// antes deste e não tem como saber — ver audit/ator.go.
			audit.AtorFromContext(r.Context()).Registrar(claims.UserID, claims.Email, claims.Role)

			// Vale para o SSE também: um stream aberto é acesso como qualquer
			// outro, e ficaria fora da trava se a checagem morasse só acima.
			if bloqueadoPorSenhaProvisoria(claims, r.URL.Path) {
				response.Forbidden(w, "troque a senha provisória antes de continuar")
				return
			}

			ctx := context.WithValue(r.Context(), CtxKeyUserClaims, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole garante que o usuário autenticado possui uma das roles permitidas.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				response.Unauthorized(w, "não autenticado")
				return
			}
			if _, ok := allowed[claims.Role]; !ok {
				response.Forbidden(w, "acesso negado")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// MembroChecker responde se um usuário pertence a um grupo.
// internal/auth.Repository já satisfaz a interface.
type MembroChecker interface {
	ValidateUsuarioGrupo(ctx context.Context, usuarioID, grupoID string) (bool, error)
}

/*
RequireGrupoMembro garante que o usuário pertence ao grupo da rota, extraído do
path param "grupoID".

A versão anterior comparava apenas claims.GrupoID com o grupo da URL, sem nunca
consultar usuario_grupos — o nome prometia uma verificação que não existia. A
diferença prática é a revogação: tirar alguém de um grupo não tinha efeito
nenhum até o token expirar, porque o grupo viajava dentro do próprio token.

As duas condições valem juntas, e cada uma cobre uma coisa:

  - o grupo do token precisa bater com o da URL — é o contexto ativo da sessão,
    e quem está em dois grupos não opera no B enquanto entrou no A;
  - o vínculo precisa existir no banco agora, não no momento em que o token foi
    emitido.

admin_global passa direto: é privilégio de plataforma, e o próprio token não
carrega grupo.

`membros` nil mantém só a comparação do token — é o comportamento antigo, para
montagens de teste que não têm banco. Nunca afrouxa em relação ao que havia.
*/
func RequireGrupoMembro(membros MembroChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				response.Unauthorized(w, "não autenticado")
				return
			}

			if claims.Role == "admin_global" {
				next.ServeHTTP(w, r)
				return
			}

			grupoID := chi.URLParam(r, "grupoID")
			if grupoID == "" {
				// Rota sem grupo no path — nada a verificar aqui.
				next.ServeHTTP(w, r)
				return
			}

			if claims.GrupoID != grupoID {
				response.Forbidden(w, "acesso negado a este grupo")
				return
			}

			if membros == nil {
				next.ServeHTTP(w, r)
				return
			}

			pertence, err := membros.ValidateUsuarioGrupo(r.Context(), claims.UserID, grupoID)
			if err != nil {
				// Falha ao consultar fecha a porta: liberar em caso de erro
				// transformaria uma indisponibilidade do banco em acesso livre.
				response.Error(w, http.StatusInternalServerError, "erro ao verificar vínculo com o grupo", err)
				return
			}
			if !pertence {
				response.Forbidden(w, "acesso negado a este grupo")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ClaimsFromContext extrai as claims do contexto.
func ClaimsFromContext(ctx context.Context) (*JWTClaims, bool) {
	claims, ok := ctx.Value(CtxKeyUserClaims).(*JWTClaims)
	return claims, ok
}

func extractBearer(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if strings.HasPrefix(header, "Bearer ") {
		return strings.TrimPrefix(header, "Bearer ")
	}
	return ""
}
