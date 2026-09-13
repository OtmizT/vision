-- Migration 000033: configuracao do assistente de IA.
--
-- Duas tabelas, porque sao duas decisoes de natureza diferente:
--
--   ia_config  COMO falar com o provedor. Uma linha so, da plataforma inteira:
--              o admin global cadastra a credencial e a OTM paga a conta.
--   ia_grupos  QUAIS grupos tem o recurso ligado. Habilitar um grupo significa
--              que os dados financeiros daquele cliente passam a sair para um
--              provedor externo, entao a decisao e por grupo e fica registrada
--              com data e autor.

CREATE TABLE IF NOT EXISTS _etl.ia_config (
    -- Linha unica. O CHECK e o que impede uma segunda configuracao aparecer e
    -- o sistema passar a depender de qual delas foi lida primeiro.
    id               INT         PRIMARY KEY DEFAULT 1 CHECK (id = 1),

    provedor         TEXT        NOT NULL DEFAULT 'deepseek',
    modelo           TEXT        NOT NULL DEFAULT 'deepseek-v4-flash',
    -- Vazio usa o padrao do provedor. Existe para permitir trocar de provedor
    -- compativel com OpenAI sem mexer em codigo.
    base_url         TEXT        NOT NULL DEFAULT '',

    -- Em texto puro, seguindo o precedente de _etl.empresas.app_secret. A
    -- saida da API e mascarada (ver internal/ia_config/types.go) e o campo e
    -- redigido na auditoria. Cifrar esta e a da Omie de uma vez e pendencia
    -- registrada; cifrar so esta, com sessenta secrets em texto puro na tabela
    -- ao lado, seria teatro.
    api_key          TEXT        NOT NULL DEFAULT '',

    max_tokens       INT         NOT NULL DEFAULT 2048 CHECK (max_tokens > 0),
    -- Teto diario de tokens por usuario. Chamada de modelo custa dinheiro e uma
    -- pergunta em loop nao pode virar fatura.
    teto_tokens_dia  INT         NOT NULL DEFAULT 200000 CHECK (teto_tokens_dia > 0),

    -- Desliga o recurso inteiro sem apagar a credencial.
    ativo            BOOLEAN     NOT NULL DEFAULT false,

    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by       UUID        REFERENCES _etl.usuarios(id)
);

-- A linha existe desde o inicio, desativada e sem credencial. Assim o GET da
-- tela de configuracao sempre encontra algo para mostrar, e o service nunca
-- precisa tratar "configuracao ausente" como caso separado de "desligada".
INSERT INTO _etl.ia_config (id) VALUES (1) ON CONFLICT (id) DO NOTHING;


CREATE TABLE IF NOT EXISTS _etl.ia_grupos (
    grupo_id    UUID        PRIMARY KEY REFERENCES _etl.grupos(id) ON DELETE CASCADE,
    ativa       BOOLEAN     NOT NULL DEFAULT false,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by  UUID        REFERENCES _etl.usuarios(id)
);

-- Sem seed, e de proposito: AUSENCIA DE LINHA SIGNIFICA DESLIGADO.
--
-- E o que faz "grupo novo nasce desligado" ser verdade estrutural, em vez de
-- depender de alguem lembrar de inserir a linha certa ao provisionar um tenant.
-- Um grupo so aparece aqui quando alguem decidiu ligar (ou desligar) o recurso
-- para ele, e a linha carrega quem decidiu e quando.
