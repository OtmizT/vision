-- Volta a coluna. As sessoes revogadas pela up nao sao restauradas: elas nao
-- tem contexto, e e exatamente por isso que foram encerradas.
ALTER TABLE _etl.refresh_tokens
    DROP CONSTRAINT IF EXISTS refresh_tokens_contexto_check;

ALTER TABLE _etl.refresh_tokens
    DROP COLUMN IF EXISTS contexto;
