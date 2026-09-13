-- ============================================================
-- Configuração do assistente de IA
-- ============================================================

-- name: GetIAConfig :one
SELECT c.id, c.provedor, c.modelo, c.base_url, c.api_key, c.max_tokens,
       c.teto_tokens_dia, c.ativo, c.updated_at, c.updated_by,
       u.email AS updated_by_email
FROM _etl.ia_config c
LEFT JOIN _etl.usuarios u ON u.id = c.updated_by
WHERE c.id = 1;

-- name: UpdateIAConfig :exec
UPDATE _etl.ia_config
SET provedor        = $1,
    modelo          = $2,
    base_url        = $3,
    api_key         = $4,
    max_tokens      = $5,
    teto_tokens_dia = $6,
    ativo           = $7,
    updated_at      = NOW(),
    updated_by      = $8
WHERE id = 1;

-- ============================================================
-- Liga/desliga por grupo
-- ============================================================

-- name: ListIAGrupos :many
-- Todos os grupos, com o estado do recurso. LEFT JOIN porque ausência de linha
-- em ia_grupos significa desligado — a tela precisa listar inclusive os grupos
-- que ninguém tocou ainda.
SELECT g.id                        AS grupo_id,
       g.nome                      AS grupo_nome,
       g.slug                      AS grupo_slug,
       COALESCE(ig.ativa, false)   AS ativa,
       ig.updated_at,
       u.email                     AS updated_by_email
FROM _etl.grupos g
LEFT JOIN _etl.ia_grupos ig ON ig.grupo_id = g.id
LEFT JOIN _etl.usuarios  u  ON u.id = ig.updated_by
WHERE g.deleted_at IS NULL
ORDER BY g.nome;

-- name: SetIAGrupo :exec
INSERT INTO _etl.ia_grupos (grupo_id, ativa, updated_at, updated_by)
VALUES ($1, $2, NOW(), $3)
ON CONFLICT (grupo_id) DO UPDATE
SET ativa = EXCLUDED.ativa, updated_at = NOW(), updated_by = EXCLUDED.updated_by;

-- name: IAAtivaNoGrupo :one
-- Portão do chat. COALESCE porque grupo sem linha é grupo desligado.
-- O ::boolean é obrigatório: sem ele o sqlc não consegue inferir o tipo do
-- COALESCE sobre subconsulta e gera interface{}.
SELECT COALESCE(
    (SELECT ativa FROM _etl.ia_grupos WHERE grupo_id = $1),
    false
)::boolean AS ativa;

-- ============================================================
-- Conversa e mensagens
-- ============================================================

-- name: UpsertIAConversa :one
-- Devolve a conversa do par, criando na primeira mensagem. O DO UPDATE existe
-- para o RETURNING trazer a linha também quando ela já existia — sem ele o
-- ON CONFLICT DO NOTHING não retorna nada e seria preciso um SELECT extra.
INSERT INTO _etl.ia_conversas (grupo_id, usuario_id)
VALUES ($1, $2)
ON CONFLICT (grupo_id, usuario_id) DO UPDATE
SET updated_at = NOW()
RETURNING id, grupo_id, usuario_id, created_at, updated_at;

-- name: InsertIAMensagem :one
INSERT INTO _etl.ia_mensagens (conversa_id, papel, conteudo, spec, fonte, tokens)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, conversa_id, papel, conteudo, spec, fonte, tokens, created_at;

-- name: ListIAMensagens :many
-- As N mais recentes, em ordem decrescente; quem chama inverte para exibir.
-- Decrescente com LIMIT é o que permite pegar o fim de uma conversa longa sem
-- ler tudo.
SELECT id, conversa_id, papel, conteudo, spec, fonte, tokens, created_at
FROM _etl.ia_mensagens
WHERE conversa_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: DeleteIAConversa :exec
-- As mensagens somem pelo CASCADE.
DELETE FROM _etl.ia_conversas
WHERE grupo_id = $1 AND usuario_id = $2;

-- name: TokensUsadosHoje :one
-- Soma dos tokens do usuário no dia, para o teto diário. Atravessa as conversas
-- de todos os grupos: o teto é do usuário, não da conversa.
SELECT COALESCE(SUM(m.tokens), 0)::bigint AS total
FROM _etl.ia_mensagens m
JOIN _etl.ia_conversas c ON c.id = m.conversa_id
WHERE c.usuario_id = $1
  AND m.created_at >= date_trunc('day', NOW());

-- name: ExpurgarIAMensagens :execrows
-- Expurgo por idade. O histórico guarda valores financeiros de clientes em
-- texto; prazo de descarte é exigência de LGPD, não preferência de produto.
DELETE FROM _etl.ia_mensagens
WHERE created_at < NOW() - make_interval(days => $1::int);

-- name: ExpurgarIAConversasVazias :execrows
-- Conversa que perdeu todas as mensagens para o expurgo não deve sobreviver
-- como casca.
DELETE FROM _etl.ia_conversas c
WHERE NOT EXISTS (SELECT 1 FROM _etl.ia_mensagens m WHERE m.conversa_id = c.id);
