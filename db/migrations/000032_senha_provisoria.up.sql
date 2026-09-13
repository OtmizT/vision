-- Migration 000032: senha definida por administrador e provisoria.
--
-- Hoje um administrador define a senha de alguem e essa senha vale para sempre.
-- Quem definiu conhece a senha da pessoa por tempo indeterminado, e nada na
-- interface pede a troca -- so a boa vontade de quem recebeu.
--
-- A partir daqui, toda senha definida por terceiro nasce provisoria, e o
-- primeiro acesso obriga a troca antes de qualquer outra coisa.
--
-- DEFAULT false, e nao true: marcar as senhas existentes como provisorias
-- obrigaria todo mundo a trocar de senha no proximo login, sem aviso, por um
-- problema que nao e deles. A regra vale das proximas definicoes em diante.

ALTER TABLE _etl.usuarios
    ADD COLUMN IF NOT EXISTS senha_provisoria BOOLEAN NOT NULL DEFAULT false;
