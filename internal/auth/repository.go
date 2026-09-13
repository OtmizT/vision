package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"omie-sync-api/sqlc/generated"
)

type Repository interface {
	GetUsuarioByEmail(ctx context.Context, email string) (*Usuario, error)
	GetUsuarioByID(ctx context.Context, id string) (*Usuario, error)
	InsertRefreshToken(ctx context.Context, usuarioID, token string, expiresAt time.Time, grupoID string, contexto Contexto) (*RefreshToken, error)
	GetRefreshToken(ctx context.Context, token string) (*RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, token string) error
	RevokeAllUserTokens(ctx context.Context, usuarioID string) error
	GetGruposByUsuarioID(ctx context.Context, usuarioID string) ([]GrupoInfo, error)
	ValidateUsuarioGrupo(ctx context.Context, usuarioID, grupoID string) (bool, error)
	GetRoleNoGrupo(ctx context.Context, usuarioID, grupoID string) (string, error)
}

type repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &repository{pool: pool}
}

func (r *repository) GetUsuarioByEmail(ctx context.Context, email string) (*Usuario, error) {
	q := sqlcgen.New(r.pool)
	row, err := q.GetUsuarioByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("auth.repository.GetUsuarioByEmail: %w", err)
	}
	return rowToUsuario(row.ID, row.GrupoID, row.Nome, row.Email, row.Password, row.Role, row.Ativo, row.CreatedAt, row.UpdatedAt), nil
}

func (r *repository) GetUsuarioByID(ctx context.Context, id string) (*Usuario, error) {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(id); err != nil {
		return nil, fmt.Errorf("auth.repository.GetUsuarioByID scan uuid: %w", err)
	}
	row, err := q.GetUsuarioByID(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("auth.repository.GetUsuarioByID: %w", err)
	}
	return rowToUsuario(row.ID, row.GrupoID, row.Nome, row.Email, row.Password, row.Role, row.Ativo, row.CreatedAt, row.UpdatedAt), nil
}

// InsertRefreshToken grava o refresh token junto com o grupo ativo e o contexto
// da sessão. grupoID vazio é permitido (contexto de plataforma) e vira NULL.
func (r *repository) InsertRefreshToken(ctx context.Context, usuarioID, token string, expiresAt time.Time, grupoID string, contexto Contexto) (*RefreshToken, error) {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(usuarioID); err != nil {
		return nil, fmt.Errorf("auth.repository.InsertRefreshToken scan uuid: %w", err)
	}
	var exp pgtype.Timestamptz
	if err := exp.Scan(expiresAt); err != nil {
		return nil, fmt.Errorf("auth.repository.InsertRefreshToken scan expires_at: %w", err)
	}
	var gid pgtype.UUID
	if grupoID != "" {
		if err := gid.Scan(grupoID); err != nil {
			return nil, fmt.Errorf("auth.repository.InsertRefreshToken scan grupo_id: %w", err)
		}
	}
	row, err := q.InsertRefreshToken(ctx, sqlcgen.InsertRefreshTokenParams{
		UsuarioID: uid,
		Token:     token,
		ExpiresAt: exp,
		GrupoID:   gid,
		Contexto:  string(contexto),
	})
	if err != nil {
		return nil, fmt.Errorf("auth.repository.InsertRefreshToken: %w", err)
	}
	return &RefreshToken{
		ID:        uuidToStr(row.ID),
		UsuarioID: uuidToStr(row.UsuarioID),
		Token:     row.Token,
		ExpiresAt: row.ExpiresAt.Time,
		Revoked:   row.Revoked,
		CreatedAt: row.CreatedAt.Time,
		GrupoID:   uuidToStr(row.GrupoID),
		Contexto:  Contexto(row.Contexto),
	}, nil
}

func (r *repository) GetRefreshToken(ctx context.Context, token string) (*RefreshToken, error) {
	q := sqlcgen.New(r.pool)
	row, err := q.GetRefreshToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("auth.repository.GetRefreshToken: %w", err)
	}
	return &RefreshToken{
		ID:        uuidToStr(row.ID),
		UsuarioID: uuidToStr(row.UsuarioID),
		Token:     row.Token,
		ExpiresAt: row.ExpiresAt.Time,
		Revoked:   row.Revoked,
		CreatedAt: row.CreatedAt.Time,
		GrupoID:   uuidToStr(row.GrupoID),
		Contexto:  Contexto(row.Contexto),
	}, nil
}

func (r *repository) RevokeRefreshToken(ctx context.Context, token string) error {
	q := sqlcgen.New(r.pool)
	if err := q.RevokeRefreshToken(ctx, token); err != nil {
		return fmt.Errorf("auth.repository.RevokeRefreshToken: %w", err)
	}
	return nil
}

func (r *repository) RevokeAllUserTokens(ctx context.Context, usuarioID string) error {
	q := sqlcgen.New(r.pool)
	var uid pgtype.UUID
	if err := uid.Scan(usuarioID); err != nil {
		return fmt.Errorf("auth.repository.RevokeAllUserTokens scan uuid: %w", err)
	}
	if err := q.RevokeAllUserTokens(ctx, uid); err != nil {
		return fmt.Errorf("auth.repository.RevokeAllUserTokens: %w", err)
	}
	return nil
}

func isUndefinedTable(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "42P01"
}

func isUndefinedColumn(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "42703"
}

func (r *repository) GetGruposByUsuarioID(ctx context.Context, usuarioID string) ([]GrupoInfo, error) {
	const q = `
		SELECT g.id, g.nome, g.slug, g.schema_name
		FROM _etl.grupos g
		JOIN _etl.usuario_grupos ug ON ug.grupo_id = g.id
		WHERE ug.usuario_id = $1::uuid AND g.deleted_at IS NULL
		ORDER BY g.nome`

	rows, err := r.pool.Query(ctx, q, usuarioID)
	if err != nil {
		if isUndefinedTable(err) {
			return nil, nil // migration pendente — retorna lista vazia
		}
		return nil, fmt.Errorf("auth.repository.GetGruposByUsuarioID: %w", err)
	}
	defer rows.Close()

	var grupos []GrupoInfo
	for rows.Next() {
		var g GrupoInfo
		var rawID pgtype.UUID
		if err := rows.Scan(&rawID, &g.Nome, &g.Slug, &g.SchemaName); err != nil {
			return nil, fmt.Errorf("auth.repository.GetGruposByUsuarioID scan: %w", err)
		}
		g.ID = uuidToStr(rawID)
		grupos = append(grupos, g)
	}
	return grupos, rows.Err()
}

func (r *repository) ValidateUsuarioGrupo(ctx context.Context, usuarioID, grupoID string) (bool, error) {
	const q = `SELECT COUNT(*) > 0 FROM _etl.usuario_grupos WHERE usuario_id = $1::uuid AND grupo_id = $2::uuid`
	var pertence bool
	if err := r.pool.QueryRow(ctx, q, usuarioID, grupoID).Scan(&pertence); err != nil {
		if isUndefinedTable(err) {
			return false, nil
		}
		return false, fmt.Errorf("auth.repository.ValidateUsuarioGrupo: %w", err)
	}
	return pertence, nil
}

func (r *repository) GetRoleNoGrupo(ctx context.Context, usuarioID, grupoID string) (string, error) {
	// admin_global é um privilégio de plataforma — prevalece sobre qualquer configuração de grupo.
	// A query usa CASE para garantir isso mesmo se ug.role estiver desatualizado.
	const q = `
		SELECT CASE WHEN u.role = 'admin_global' THEN 'admin_global'
		            ELSE COALESCE(ug.role, u.role)
		       END
		FROM _etl.usuario_grupos ug
		JOIN _etl.usuarios u ON u.id = ug.usuario_id
		WHERE ug.usuario_id = $1::uuid AND ug.grupo_id = $2::uuid`

	var role string
	err := r.pool.QueryRow(ctx, q, usuarioID, grupoID).Scan(&role)
	if err == nil {
		return role, nil
	}

	// Tabela ou coluna ainda não existe (migrations pendentes) — fallback direto em usuarios
	if isUndefinedTable(err) || isUndefinedColumn(err) {
		const qLegacy = `SELECT role FROM _etl.usuarios WHERE id = $1::uuid`
		if err2 := r.pool.QueryRow(ctx, qLegacy, usuarioID).Scan(&role); err2 == nil {
			return role, nil
		}
	}

	// Usuário não encontrado na junction — tenta pelo menos buscar o role global
	const qFallback = `SELECT role FROM _etl.usuarios WHERE id = $1::uuid`
	if err2 := r.pool.QueryRow(ctx, qFallback, usuarioID).Scan(&role); err2 == nil {
		return role, nil
	}

	return "viewer", nil
}

// --- helpers ---

func rowToUsuario(id, grupoID pgtype.UUID, nome, email, password, role string, ativo bool, createdAt, updatedAt pgtype.Timestamptz) *Usuario {
	return &Usuario{
		ID:        uuidToStr(id),
		GrupoID:   uuidToStr(grupoID),
		Nome:      nome,
		Email:     email,
		Password:  password,
		Role:      role,
		Ativo:     ativo,
		CreatedAt: createdAt.Time,
		UpdatedAt: updatedAt.Time,
	}
}

func uuidToStr(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", u.Bytes[0:4], u.Bytes[4:6], u.Bytes[6:8], u.Bytes[8:10], u.Bytes[10:16])
}
