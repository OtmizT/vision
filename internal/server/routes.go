package server

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/rs/zerolog"

	"omie-sync-api/internal/audit"
	"omie-sync-api/internal/auth"
	"omie-sync-api/internal/dados"
	"omie-sync-api/internal/empresas"
	"omie-sync-api/internal/grupos"
	"omie-sync-api/internal/ia"
	"omie-sync-api/internal/ia_config"
	"omie-sync-api/internal/omie_config"
	"omie-sync-api/internal/permissoes"
	"omie-sync-api/internal/query"
	syncsvc "omie-sync-api/internal/sync"
	"omie-sync-api/internal/usuarios"
)

type Dependencies struct {
	AuditRepo         audit.Repository
	AuthHandler       *auth.Handler
	GruposHandler     *grupos.Handler
	EmpresasHandler   *empresas.Handler
	SyncHandler       *syncsvc.Handler
	UsuariosHandler   *usuarios.Handler
	PermissoesHandler *permissoes.Handler
	DadosHandler      *dados.Handler
	OmieConfigHandler *omie_config.Handler
	QueryHandler      *query.Handler
	IAHandler         *ia.Handler
	IAConfigHandler   *ia_config.Handler
	// Membros verifica vínculo usuário×grupo nas rotas com {grupoID}.
	Membros auth.MembroChecker
	Logger  zerolog.Logger
}

// corsMiddleware libera o frontend local em development e o domínio configurado em production.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		allowed := os.Getenv("CORS_ORIGIN")
		if allowed == "" {
			allowed = "http://localhost:5173,http://localhost:5174,http://localhost:3000"
		}

		for _, o := range strings.Split(allowed, ",") {
			if strings.TrimSpace(o) == origin {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				break
			}
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Vary", "Origin")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func NewRouter(deps Dependencies) http.Handler {
	r := chi.NewRouter()

	// Middleware base
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)

	// Rate limit global — 300 req/min por IP (proteção contra loops acidentais)
	r.Use(httprate.LimitByIP(300, 1*time.Minute))

	// Audit — cobre TODAS as rotas sem exceção
	r.Use(audit.Middleware(deps.AuditRepo, deps.Logger))

	// Health check (sem auth)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Auth
	r.Mount("/auth", deps.AuthHandler.Routes())

	// Admin
	r.Mount("/admin/grupos", deps.GruposHandler.Routes())
	r.Mount("/admin/grupos/{grupoID}/empresas", deps.EmpresasHandler.Routes())
	r.Mount("/sync", deps.SyncHandler.Routes())
	r.Mount("/admin/grupos/{grupoID}/usuarios", deps.UsuariosHandler.Routes())
	r.Mount("/admin/permissoes", deps.PermissoesHandler.Routes())
	r.Mount("/admin/omie-config", deps.OmieConfigHandler.Routes())
	r.Mount("/admin/ia-config", deps.IAConfigHandler.Routes())

	// Assistente de IA. O rate limit por usuario e montado no proprio handler,
	// com o limitador criado uma vez no wire.
	r.Mount("/ia", deps.IAHandler.Routes())

	// Admin Sync (Global)
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireAuth(deps.AuthHandler.JWTService())) // JWTService() helper might be needed or access jwtSvc directly
		r.Use(auth.RequireRole("admin_global"))
		r.Get("/admin/sync/overview", deps.SyncHandler.AdminOverview)
		r.Get("/admin/sync/empresas", deps.SyncHandler.AdminEmpresasSync)
		r.Get("/admin/sync/jobs/ativos", deps.SyncHandler.AdminJobsAtivos)
		r.Get("/admin/sync/dlq", deps.SyncHandler.AdminDLQ)
		r.Post("/admin/sync/pages/{pageID}/retry", deps.SyncHandler.AdminRetryPage)
		r.Post("/admin/sync/jobs/{jobID}/cancelar", deps.SyncHandler.AdminCancelarJob)
		r.Post("/admin/sync/startup-recovery", deps.SyncHandler.AdminStartupRecovery)

		// Manutenção operacional de banco. São ações com alcance real — o REFRESH
		// pode segurar a view por minutos e o cancelamento derruba consultas — por
		// isso vivem em endpoints com alvo explícito, e não como SQL livre no
		// SQL Explorer. O middleware de auditoria cobre todas.
		r.Get("/admin/manutencao/consultas", deps.SyncHandler.AdminConsultasAtivas)
		r.Post("/admin/manutencao/consultas/{pid}/cancelar", deps.SyncHandler.AdminCancelarConsulta)
		r.Post("/admin/manutencao/grupos/{grupoID}/refresh-views", deps.SyncHandler.AdminRefreshViews)
	})

	// SQL Explorer
	sqlRateLimiter := httprate.NewRateLimiter(20, 1*time.Minute, httprate.WithKeyFuncs(
		func(r *http.Request) (string, error) {
			claims, ok := auth.ClaimsFromContext(r.Context())
			if !ok {
				return httprate.KeyByIP(r)
			}
			return "sql:" + claims.UserID, nil
		},
	))
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireAuth(deps.AuthHandler.JWTService()))
		r.Use(auth.RequireRole("admin_global", "admin_grupo"))
		// Sem o RequireGrupoMembro, um admin do cliente A trocava o UUID da URL
		// e rodava SELECT no schema do cliente B. O middleware inline enxerga o
		// {grupoID} porque a rota já casou quando ele roda.
		r.With(sqlRateLimiter.Handler, auth.RequireGrupoMembro(deps.Membros)).
			Post("/admin/grupos/{grupoID}/query", deps.QueryHandler.Execute)
	})

	// Dados sincronizados (leitura dos schemas tenant)
	r.Mount("/dados", deps.DadosHandler.Routes())
	// r.Mount("/admin/grupos", gruposHandler.Routes())
	// r.Mount("/admin/empresas", empresasHandler.Routes())
	// r.Mount("/admin/usuarios", usuariosHandler.Routes())
	// r.Mount("/sync", syncHandler.Routes())

	return r
}
