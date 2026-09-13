-- Derruba a configuracao do assistente. A credencial some junto, e os grupos
-- perdem o registro de quem ligou o recurso e quando.
DROP TABLE IF EXISTS _etl.ia_grupos;
DROP TABLE IF EXISTS _etl.ia_config;
