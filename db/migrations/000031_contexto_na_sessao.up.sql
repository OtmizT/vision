-- Migration 000031: a sessao passa a saber em QUE contexto foi aberta.
--
-- Ate aqui o sistema so sabia "que grupo esta ativo". Quem administra a
-- plataforma e quem trabalha dentro de um cliente eram a mesma coisa, resolvida
-- por um CASE WHEN que fazia admin_global vencer o papel de grupo sempre -- e
-- por isso o admin global ficava preso fora das telas de grupo.
--
-- 'plataforma' e 'grupo' passam a ser estados distintos e explicitos. O contexto
-- viaja no access token (claim `contexto`) e aqui, no refresh token, porque
-- /auth/refresh recebe so o token opaco: sem persistir, a renovacao silenciosa
-- devolveria a pessoa a um contexto que ela nao escolheu.
--
-- A coluna existe separada de grupo_id porque grupo_id NULL e ambiguo: hoje
-- significa tanto "admin global sem grupo proprio" quanto "sessao anterior a
-- migration 000029". Deduzir contexto dali reintroduziria a ambiguidade que esta
-- migration existe para acabar.

ALTER TABLE _etl.refresh_tokens
    ADD COLUMN IF NOT EXISTS contexto TEXT NOT NULL DEFAULT 'grupo';

ALTER TABLE _etl.refresh_tokens
    DROP CONSTRAINT IF EXISTS refresh_tokens_contexto_check;

ALTER TABLE _etl.refresh_tokens
    ADD CONSTRAINT refresh_tokens_contexto_check
    CHECK (contexto IN ('plataforma', 'grupo'));

-- As sessoes em curso sao encerradas, como na 000029.
--
-- Nao ha como migra-las: o contexto delas nunca existiu, e o DEFAULT 'grupo'
-- seria um palpite -- errado justamente para o admin global, o unico usuario
-- para quem a distincao muda alguma coisa.
--
-- Os access tokens ja emitidos tambem param de valer, porque a validacao passa a
-- exigir a claim `contexto`. Quem estiver logado precisa autenticar de novo; a
-- sessao nova nasce com o contexto escolhido na tela.
UPDATE _etl.refresh_tokens
SET revoked = true
WHERE revoked = false;
