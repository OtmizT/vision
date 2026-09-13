-- name: InsertUsuario :one
INSERT INTO _etl.usuarios (grupo_id, nome, email, password, role)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, grupo_id, nome, email, role, ativo, created_at, updated_at;

-- name: GetUsuarioByIDFull :one
SELECT id, grupo_id, nome, email, role, ativo, created_at, updated_at
FROM _etl.usuarios
WHERE id = $1
  AND deleted_at IS NULL;

-- name: ListUsuariosByGrupo :many
SELECT id, grupo_id, nome, email, role, ativo, created_at, updated_at
FROM _etl.usuarios
WHERE grupo_id = $1
  AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountUsuariosByGrupo :one
SELECT COUNT(*) FROM _etl.usuarios
WHERE grupo_id = $1
  AND deleted_at IS NULL;

-- name: UpdateUsuario :one
UPDATE _etl.usuarios
SET nome = $2, role = $3, ativo = $4, updated_at = NOW()
WHERE id = $1
  AND deleted_at IS NULL
RETURNING id, grupo_id, nome, email, role, ativo, created_at, updated_at;

-- name: UpdateUsuarioPassword :exec
-- Senha definida por administrador nasce provisoria: quem definiu conhece a
-- senha da pessoa, e o primeiro acesso obriga a troca.
UPDATE _etl.usuarios
SET password = $2, senha_provisoria = true, updated_at = NOW()
WHERE id = $1;

-- name: SoftDeleteUsuario :exec
UPDATE _etl.usuarios
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: InsertUsuarioGrupo :exec
INSERT INTO _etl.usuario_grupos (usuario_id, grupo_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: DeleteUsuarioGrupo :exec
DELETE FROM _etl.usuario_grupos
WHERE usuario_id = $1 AND grupo_id = $2;

-- name: GetGruposByUsuario :many
SELECT g.id, g.nome, g.slug, g.schema_name
FROM _etl.grupos g
JOIN _etl.usuario_grupos ug ON ug.grupo_id = g.id
WHERE ug.usuario_id = $1
  AND g.deleted_at IS NULL
ORDER BY g.nome;

-- name: ListUsuariosByGrupoFromJunction :many
SELECT u.id, u.grupo_id, u.nome, u.email, u.role, u.ativo, u.created_at, u.updated_at
FROM _etl.usuarios u
JOIN _etl.usuario_grupos ug ON ug.usuario_id = u.id
WHERE ug.grupo_id = $1
  AND u.deleted_at IS NULL
ORDER BY u.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountUsuariosByGrupoFromJunction :one
SELECT COUNT(*)
FROM _etl.usuarios u
JOIN _etl.usuario_grupos ug ON ug.usuario_id = u.id
WHERE ug.grupo_id = $1
  AND u.deleted_at IS NULL;
