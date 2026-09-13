-- Migration 000034: historico do chat com o assistente.
--
-- Uma conversa continua por par usuario x grupo, e nao threads separadas: a
-- pessoa abre o painel e continua de onde parou. Threads ficam para depois, se
-- a necessidade aparecer.
--
-- O par usuario x grupo, e nao so usuario, porque a mesma pessoa pode
-- administrar dois clientes. Misturar as duas conversas mostraria numero de um
-- cliente no contexto do outro -- exatamente o que o resto do sistema passa o
-- tempo todo impedindo.

CREATE TABLE IF NOT EXISTS _etl.ia_conversas (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    grupo_id    UUID        NOT NULL REFERENCES _etl.grupos(id)   ON DELETE CASCADE,
    usuario_id  UUID        NOT NULL REFERENCES _etl.usuarios(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Avanca a cada mensagem nova. E por ele que o expurgo decide o que sumiu
    -- de uso, e nao pelo created_at: uma conversa de janeiro ainda ativa em
    -- setembro nao deve ser apagada.
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Uma conversa por par. O upsert do repositorio depende desta restricao.
    UNIQUE (grupo_id, usuario_id)
);

CREATE INDEX IF NOT EXISTS idx_ia_conversas_updated
    ON _etl.ia_conversas (updated_at DESC);


CREATE TABLE IF NOT EXISTS _etl.ia_mensagens (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    conversa_id UUID        NOT NULL REFERENCES _etl.ia_conversas(id) ON DELETE CASCADE,

    papel       TEXT        NOT NULL CHECK (papel IN ('usuario', 'assistente')),
    -- Markdown, nao HTML. O frontend converte e sanitiza na exibicao; guardar
    -- HTML aqui faria o banco carregar a superficie de XSS junto com o dado.
    conteudo    TEXT        NOT NULL,

    -- Especificacao do grafico, quando a resposta produziu um. NAO e config do
    -- Chart.js: e a descricao do que desenhar, que o frontend converte usando a
    -- paleta e as fontes da aplicacao.
    spec        JSONB,
    -- Qual ferramenta e quais filtros produziram o numero. E o "mostre a conta"
    -- que faz alguem do financeiro confiar na resposta.
    fonte       JSONB,

    tokens      INT         NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Composto porque toda leitura e "as mensagens desta conversa, em ordem".
CREATE INDEX IF NOT EXISTS idx_ia_mensagens_conversa
    ON _etl.ia_mensagens (conversa_id, created_at);

-- O expurgo varre por data pura, sem conversa.
CREATE INDEX IF NOT EXISTS idx_ia_mensagens_created
    ON _etl.ia_mensagens (created_at);
