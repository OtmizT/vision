-- So a restricao volta atras. O rebaixamento nao e revertido: nao ha registro
-- de quais linhas diziam 'admin_global' antes, e recriar o valor a partir de
-- usuarios.role reintroduziria exatamente a escalada que a up fechou.
ALTER TABLE _etl.usuario_grupos
    DROP CONSTRAINT IF EXISTS usuario_grupos_role_check;
