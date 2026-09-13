# Emergência: recuperar o acesso do admin global

Existe **um** admin global. Não há um segundo administrador para socorrer, e
nenhuma tela concede o papel — por desenho, desde a migration 000030. Se o
acesso dele se perder, o caminho é este arquivo e acesso direto ao banco.

Rode contra o banco de produção, no schema `_etl`.

## 1. Diagnóstico — o que está gravado

```sql
SELECT u.id, u.email, u.role AS papel_global, u.ativo
FROM _etl.usuarios u
WHERE u.role = 'admin_global' AND u.deleted_at IS NULL;
```

Zero linhas é o caso grave: ninguém administra a plataforma. Mais de uma linha
contraria a regra de "um admin global" — investigar antes de mexer.

```sql
-- Papel dentro de cada grupo e sessões vivas
SELECT g.nome, ug.role
FROM _etl.usuario_grupos ug
JOIN _etl.grupos g ON g.id = ug.grupo_id
WHERE ug.usuario_id = '<uuid>';

SELECT count(*) FROM _etl.refresh_tokens
WHERE usuario_id = '<uuid>' AND revoked = false AND expires_at > NOW();
```

## 2. Restaurar o papel de plataforma

O poder de plataforma mora **só** em `usuarios.role`. `usuario_grupos.role` não
concede nada e não aceita `admin_global` (CHECK da 000030) — não tente por lá.

```sql
UPDATE _etl.usuarios
SET role = 'admin_global', ativo = true, updated_at = NOW()
WHERE email = '<email do admin>' AND deleted_at IS NULL;
```

## 3. Se o problema for a senha

Gerar o hash **fora do banco** e colar o resultado. Nunca escrever a senha em
claro numa query — ela fica no histórico do psql e nos logs do servidor.

```bash
# bcrypt, custo padrão (10), o mesmo que a aplicação usa
htpasswd -bnBC 10 "" 'SENHA-NOVA' | tr -d ':\n'
```

```sql
UPDATE _etl.usuarios
SET password = '<hash $2a$10$...>', updated_at = NOW()
WHERE email = '<email do admin>';

-- Derruba as sessões antigas: trocar a senha sem isso deixa quem tinha a
-- anterior navegando por até sete dias.
UPDATE _etl.refresh_tokens SET revoked = true WHERE usuario_id = '<uuid>';
```

> O hash semeado pela migration `000022_seed_admin_user.up.sql` está **em claro
> no repositório**. Com um único admin global, essa conta é a chave da
> plataforma inteira. Trocar a senha dela é pendência aberta.

## 4. Se o contexto for o problema (a partir da 000031)

O contexto viaja no refresh token e na claim `contexto` do JWT. Um token preso
em `grupo` não dá acesso às telas de plataforma. Não há o que consertar no
banco: basta **sair e entrar de novo** — a tela de escolha oferece Plataforma
para quem tem `usuarios.role = 'admin_global'`.

Para forçar, revogando tudo:

```sql
UPDATE _etl.refresh_tokens SET revoked = true WHERE usuario_id = '<uuid>';
```

## 5. Conferir que voltou

```sql
SELECT email, role, ativo FROM _etl.usuarios WHERE role = 'admin_global';
```

Depois: login pela tela, escolher **Plataforma**, e confirmar que
`/admin/sync-control` abre.
