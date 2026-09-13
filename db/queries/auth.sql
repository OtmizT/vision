-- name: GetUsuarioByEmail :one
SELECT id, grupo_id, nome, email, password, role, ativo, senha_provisoria, created_at, updated_at
FROM _etl.usuarios
WHERE email = $1
  AND deleted_at IS NULL;

-- name: GetUsuarioByID :one
SELECT id, grupo_id, nome, email, password, role, ativo, senha_provisoria, created_at, updated_at
FROM _etl.usuarios
WHERE id = $1
  AND deleted_at IS NULL;

-- name: InsertRefreshToken :one
INSERT INTO _etl.refresh_tokens (usuario_id, token, expires_at, grupo_id, contexto)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetRefreshToken :one
SELECT id, usuario_id, token, expires_at, revoked, created_at, grupo_id, contexto
FROM _etl.refresh_tokens
WHERE token = $1
  AND revoked = false
  AND expires_at > NOW();

-- name: RevokeRefreshToken :exec
UPDATE _etl.refresh_tokens
SET revoked = true
WHERE token = $1;

-- name: RevokeAllUserTokens :exec
UPDATE _etl.refresh_tokens
SET revoked = true
WHERE usuario_id = $1;

-- name: GetGruposByUsuarioID :many
SELECT g.id, g.nome, g.slug, g.schema_name
FROM _etl.grupos g
JOIN _etl.usuario_grupos ug ON ug.grupo_id = g.id
WHERE ug.usuario_id = $1
  AND g.deleted_at IS NULL
ORDER BY g.nome;

-- name: ValidateUsuarioGrupo :one
SELECT COUNT(*) > 0 AS pertence
FROM _etl.usuario_grupos
WHERE usuario_id = $1
  AND grupo_id = $2;

-- name: UpdateSenhaPropria :exec
-- Troca feita pelo proprio usuario: a senha deixa de ser provisoria.
UPDATE _etl.usuarios
SET password = $2, senha_provisoria = false, updated_at = NOW()
WHERE id = $1
  AND deleted_at IS NULL;
