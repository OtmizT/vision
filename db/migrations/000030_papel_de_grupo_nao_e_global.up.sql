-- Migration 000030: papel de grupo deixa de poder valer 'admin_global'.
--
-- admin_global e um privilegio de plataforma. Ele nunca deveria ter existido
-- como valor de usuario_grupos.role, que descreve o que a pessoa faz DENTRO de
-- um grupo -- e a coluna nasceu sem CHECK nenhum (migrations 000024 e 000025),
-- aceitando qualquer texto.
--
-- Pior: a 000025 copiou usuarios.role para usuario_grupos.role em massa, entao
-- o admin global de hoje ja tem 'admin_global' gravado no vinculo de todos os
-- grupos dele. Enquanto a role do token sai do CASE WHEN em
-- auth/repository.go, esse valor e ignorado; no momento em que ela passar a
-- sair do vinculo (fase C do plano), ele viraria escalada de privilegio.
--
-- Rebaixar aqui e o que faz valer no banco a regra de que existe UM admin
-- global. A conta dele nao perde nada: o poder de plataforma mora em
-- usuarios.role, que esta intocada. O que ele perde e um papel de grupo
-- redundante, e mesmo esse continua irrestrito pelo bypass de admin_global no
-- RequireGrupoMembro.

UPDATE _etl.usuario_grupos
SET role = 'admin_grupo'
WHERE role = 'admin_global';

-- Qualquer valor fora da lista tambem cai para viewer antes do CHECK: a coluna
-- aceitou texto livre por seis migrations, e um CHECK que falha em produzir
-- deixa a migration presa no meio.
UPDATE _etl.usuario_grupos
SET role = 'viewer'
WHERE role NOT IN ('admin_grupo', 'viewer');

ALTER TABLE _etl.usuario_grupos
    DROP CONSTRAINT IF EXISTS usuario_grupos_role_check;

ALTER TABLE _etl.usuario_grupos
    ADD CONSTRAINT usuario_grupos_role_check
    CHECK (role IN ('admin_grupo', 'viewer'));
