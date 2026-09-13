package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	rootdb "omie-sync-api/db"
	"omie-sync-api/internal/audit"
	"omie-sync-api/internal/auth"
	"omie-sync-api/internal/config"
	"omie-sync-api/internal/dados"
	"omie-sync-api/internal/db"
	"omie-sync-api/internal/empresas"
	"omie-sync-api/internal/etl"
	"omie-sync-api/internal/etl/progress"
	"omie-sync-api/internal/grupos"
	"omie-sync-api/internal/logger"
	"omie-sync-api/internal/omie_config"
	"omie-sync-api/internal/permissoes"
	"omie-sync-api/internal/query"
	"omie-sync-api/internal/server"
	syncsvc "omie-sync-api/internal/sync"
	"omie-sync-api/internal/usuarios"
	"omie-sync-api/internal/webhooks"
	"omie-sync-api/internal/worker"
)

func main() {
	// --- Config ---
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	// --- Logger ---
	log := logger.New(cfg.AppEnv)
	log.Info().Str("env", cfg.AppEnv).Msg("iniciando omie-sync-api")

	// --- DB Pool ---
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.NewPoolWithConcurrency(ctx, cfg.DatabaseURL, cfg.WorkerMaxConcurrent)
	if err != nil {
		log.Fatal().Err(err).Msg("falha ao conectar ao banco")
	}
	defer pool.Close()
	log.Info().Msg("conexão com banco estabelecida")

	// --- Migrations ---
	migResult := rootdb.RunMigrations(context.Background(), pool)
	log.Info().
		Strs("applied", migResult.Applied).
		Strs("failed", migResult.Failed).
		Int("skipped", len(migResult.Skipped)).
		Msg("migrations concluídas")

	// --- Repositories ---
	auditRepo := audit.NewRepository(pool)
	authRepo := auth.NewRepository(pool)
	gruposRepo := grupos.NewRepository(pool)
	empresasRepo := empresas.NewRepository(pool)
	syncRepo := syncsvc.NewRepository(pool)
	webhooksRepo := webhooks.NewRepository(pool)
	usuariosRepo := usuarios.NewRepository(pool)
	permissoesRepo := permissoes.NewRepository(pool)
	omieConfigRepo := omie_config.NewRepository(pool)

	// --- SSE Hub ---
	sseHub := syncsvc.NewSSEHub()

	// --- Provisioner ---
	provisioner := db.NewProvisioner(pool)

	// --- Services ---
	jwtSvc := auth.NewJWTService(cfg.JWTSecret)

	authSvc := auth.NewService(authRepo, jwtSvc)
	gruposSvc := grupos.NewService(gruposRepo, provisioner)
	empresasSvc := empresas.NewService(empresasRepo)
	dispatcher := webhooks.NewDispatcher(webhooksRepo, log)
	syncSvc := syncsvc.NewService(syncRepo, dispatcher, log)
	usuariosSvc := usuarios.NewService(usuariosRepo, authRepo)
	permissoesSvc := permissoes.NewService(permissoesRepo)
	omieConfigSvc := omie_config.NewService(omieConfigRepo)

	// --- Handlers ---
	authHandler := auth.NewHandler(authSvc, jwtSvc)
	gruposHandler := grupos.NewHandler(gruposSvc, jwtSvc, authRepo)
	empresasHandler := empresas.NewHandler(empresasSvc, jwtSvc, authRepo)
	syncHandler := syncsvc.NewHandler(syncSvc, jwtSvc, sseHub)
	usuariosHandler := usuarios.NewHandler(usuariosSvc, jwtSvc, authRepo)
	permissoesHandler := permissoes.NewHandler(permissoesSvc, jwtSvc)
	dadosHandler := dados.NewHandler(pool, jwtSvc)
	omieConfigHandler := omie_config.NewHandler(omieConfigSvc, jwtSvc)
	querySvc := query.NewService()
	queryHandler := query.NewHandler(querySvc, pool, jwtSvc)

	// --- ETL Worker + Scheduler ---
	executors := etl.NewAllExecutors(pool, log)
	fetcher := worker.NewEmpresaFetcher(pool)
	reporter := progress.NewDBReporter(pool, syncRepo)
	etlWorker := worker.NewWorker(syncRepo, fetcher, executors, dispatcher, reporter, omieConfigSvc, sseHub, pool, log)
	scheduler := worker.NewScheduler(pool, syncRepo, etlWorker, cfg.WorkerMaxConcurrent, log)

	// Registra o worker e o pool de concorrência no syncSvc
	syncsvc.SetProcessor(syncSvc, etlWorker)
	syncsvc.SetSubmitter(syncSvc, scheduler.Pool())

	// Recupera jobs que ficaram presos em caso de reinício inesperado
	if err := syncSvc.StartupRecovery(context.Background()); err != nil {
		log.Error().Err(err).Msg("falha no startup recovery — continuando mesmo assim")
	}

	// Provisiona os schemas ANTES de o scheduler subir.
	//
	// Antes isso só acontecia no início de um job de sync, o que criava um impasse:
	// o upgrade precisa dropar as views materializadas, o DROP precisa do lock
	// exclusivo, e o lock costuma estar ocupado justamente pelo REFRESH lento que o
	// upgrade viria corrigir. O job travava e a correção nunca era aplicada.
	//
	// No processo recém-iniciado não há refresh em andamento, então o lock está livre.
	provisionarSchemas(context.Background(), pool, provisioner, log)

	scheduler.Start()
	defer scheduler.Stop()

	// --- Background Jobs ---
	deletionJob := empresas.NewDeletionJob(empresasRepo, log)
	deletionJob.Start()
	defer deletionJob.Stop()

	// --- Router ---
	router := server.NewRouter(server.Dependencies{
		AuditRepo:         auditRepo,
		AuthHandler:       authHandler,
		GruposHandler:     gruposHandler,
		EmpresasHandler:   empresasHandler,
		SyncHandler:       syncHandler,
		UsuariosHandler:   usuariosHandler,
		PermissoesHandler: permissoesHandler,
		DadosHandler:      dadosHandler,
		OmieConfigHandler: omieConfigHandler,
		QueryHandler:      queryHandler,
		Membros:           authRepo,
		Logger:            log,
	})

	// --- HTTP Server ---
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	done := make(chan struct{})
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		sig := <-quit
		log.Info().Str("signal", sig.String()).Msg("sinal de shutdown recebido")

		shutCtx, shutCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutCancel()

		if err := srv.Shutdown(shutCtx); err != nil {
			log.Error().Err(err).Msg("erro no shutdown do servidor")
		}
		close(done)
	}()

	log.Info().Str("port", cfg.Port).Msg("servidor iniciado")

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("erro ao iniciar servidor")
	}

	<-done
	log.Info().Msg("servidor encerrado")
}

// provisionarSchemas aplica upgrades pendentes de schema em todos os grupos ativos.
//
// Roda no startup, de propósito. Enquanto isso dependia do início de um job de sync,
// um schema desatualizado só era corrigido se alguém sincronizasse — e, pior, o DROP
// das views materializadas disputava lock com o REFRESH que o upgrade viria consertar.
//
// Falha de um grupo não impede os demais nem a subida da aplicação: sem isso, um
// schema problemático deixaria a API inteira fora do ar.
func provisionarSchemas(ctx context.Context, pool *pgxpool.Pool, provisioner *db.Provisioner, log zerolog.Logger) {
	rows, err := pool.Query(ctx,
		`SELECT schema_name FROM _etl.grupos WHERE deleted_at IS NULL ORDER BY nome`)
	if err != nil {
		log.Error().Err(err).Msg("provisionamento: falha ao listar grupos — continuando")
		return
	}
	var schemas []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			log.Error().Err(err).Msg("provisionamento: falha ao ler grupo")
			continue
		}
		schemas = append(schemas, s)
	}
	rows.Close()

	atualizados := 0
	for _, schema := range schemas {
		if !provisioner.NeedsProvisioning(ctx, schema) {
			continue
		}
		inicio := time.Now()
		if err := provisioner.ProvisionSchema(ctx, schema); err != nil {
			log.Error().Err(err).Str("schema", schema).
				Msg("provisionamento: falha no upgrade de schema — será tentado no próximo sync")
			continue
		}
		atualizados++
		log.Info().Str("schema", schema).Dur("duracao", time.Since(inicio)).
			Msg("provisionamento: schema atualizado")
	}

	log.Info().Int("grupos", len(schemas)).Int("atualizados", atualizados).
		Msg("provisionamento de schemas concluído")
}
