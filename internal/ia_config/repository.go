package ia_config

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	sqlcgen "omie-sync-api/sqlc/generated"
)

type Repository interface {
	Get(ctx context.Context) (*Config, error)
	Update(ctx context.Context, req UpdateRequest, apiKey, usuarioID string) error
	ListGrupos(ctx context.Context) ([]GrupoIA, error)
	SetGrupo(ctx context.Context, grupoID string, ativa bool, usuarioID string) error
	AtivaNoGrupo(ctx context.Context, grupoID string) (bool, error)
}

type repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &repository{pool: pool}
}

func (r *repository) Get(ctx context.Context) (*Config, error) {
	q := sqlcgen.New(r.pool)
	row, err := q.GetIAConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("ia_config.repository.Get: %w", err)
	}
	return &Config{
		Provedor:       row.Provedor,
		Modelo:         row.Modelo,
		BaseURL:        row.BaseUrl,
		APIKey:         row.ApiKey,
		MaxTokens:      row.MaxTokens,
		TetoTokensDia:  row.TetoTokensDia,
		Ativo:          row.Ativo,
		UpdatedAt:      row.UpdatedAt.Time,
		UpdatedByEmail: row.UpdatedByEmail.String,
	}, nil
}

// Update grava a configuração. apiKey já vem resolvida pelo service — o
// repositório não decide se preserva ou substitui a chave.
func (r *repository) Update(ctx context.Context, req UpdateRequest, apiKey, usuarioID string) error {
	q := sqlcgen.New(r.pool)

	var uid pgtype.UUID
	if usuarioID != "" {
		if err := uid.Scan(usuarioID); err != nil {
			return fmt.Errorf("ia_config.repository.Update scan usuario: %w", err)
		}
	}

	err := q.UpdateIAConfig(ctx, sqlcgen.UpdateIAConfigParams{
		Provedor:      req.Provedor,
		Modelo:        req.Modelo,
		BaseUrl:       req.BaseURL,
		ApiKey:        apiKey,
		MaxTokens:     req.MaxTokens,
		TetoTokensDia: req.TetoTokensDia,
		Ativo:         req.Ativo,
		UpdatedBy:     uid,
	})
	if err != nil {
		return fmt.Errorf("ia_config.repository.Update: %w", err)
	}
	return nil
}

func (r *repository) ListGrupos(ctx context.Context) ([]GrupoIA, error) {
	q := sqlcgen.New(r.pool)
	rows, err := q.ListIAGrupos(ctx)
	if err != nil {
		return nil, fmt.Errorf("ia_config.repository.ListGrupos: %w", err)
	}

	out := make([]GrupoIA, 0, len(rows))
	for _, row := range rows {
		out = append(out, GrupoIA{
			GrupoID:        uuidToStr(row.GrupoID),
			GrupoNome:      row.GrupoNome,
			GrupoSlug:      row.GrupoSlug,
			Ativa:          row.Ativa,
			UpdatedAt:      row.UpdatedAt.Time,
			UpdatedByEmail: row.UpdatedByEmail.String,
		})
	}
	return out, nil
}

func (r *repository) SetGrupo(ctx context.Context, grupoID string, ativa bool, usuarioID string) error {
	q := sqlcgen.New(r.pool)

	var gid pgtype.UUID
	if err := gid.Scan(grupoID); err != nil {
		return fmt.Errorf("ia_config.repository.SetGrupo scan grupo: %w", err)
	}
	var uid pgtype.UUID
	if usuarioID != "" {
		if err := uid.Scan(usuarioID); err != nil {
			return fmt.Errorf("ia_config.repository.SetGrupo scan usuario: %w", err)
		}
	}

	if err := q.SetIAGrupo(ctx, sqlcgen.SetIAGrupoParams{
		GrupoID:   gid,
		Ativa:     ativa,
		UpdatedBy: uid,
	}); err != nil {
		return fmt.Errorf("ia_config.repository.SetGrupo: %w", err)
	}
	return nil
}

// AtivaNoGrupo é o portão do chat. Grupo sem linha é grupo desligado — o
// COALESCE da query cuida disso, e é por isso que não há erro de "não
// encontrado" aqui.
func (r *repository) AtivaNoGrupo(ctx context.Context, grupoID string) (bool, error) {
	q := sqlcgen.New(r.pool)

	var gid pgtype.UUID
	if err := gid.Scan(grupoID); err != nil {
		return false, fmt.Errorf("ia_config.repository.AtivaNoGrupo scan grupo: %w", err)
	}

	ativa, err := q.IAAtivaNoGrupo(ctx, gid)
	if err != nil {
		return false, fmt.Errorf("ia_config.repository.AtivaNoGrupo: %w", err)
	}
	return ativa, nil
}

func uuidToStr(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", u.Bytes[0:4], u.Bytes[4:6], u.Bytes[6:8], u.Bytes[8:10], u.Bytes[10:16])
}
