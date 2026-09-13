package audit

import "context"

/*
Quem fez a requisição.

A tabela audit_logs tem user_id, user_email e role desde a migration 000007, e
nenhum dos três era preenchido: o middleware de auditoria roda na raiz do router
(regra 4 do CLAUDE.md — cobre TODAS as rotas) e a autenticação roda dentro de
cada subrouter, depois. Quando o middleware montava a LogEntry, ninguém havia
validado o token ainda.

A saída não é inverter a ordem — isso quebraria a cobertura total, porque as
rotas públicas não passam por autenticação nenhuma. O middleware põe no contexto
um Ator vazio e o RequireAuth o preenche ao validar o token. Como é um ponteiro,
a escrita feita lá dentro é visível aqui fora, mesmo o r.WithContext tendo criado
outro Request.

Sem isso a trilha de auditoria registra o que aconteceu e não registra quem fez —
que é a metade que importa quando se investiga um acesso indevido.
*/
type Ator struct {
	UserID string
	Email  string
	Role   string
}

const CtxKeyAtor contextKey = "audit_ator"

// ComAtor devolve um contexto carregando um Ator vazio, e o próprio Ator para
// quem for preenchê-lo depois.
func ComAtor(ctx context.Context) (context.Context, *Ator) {
	a := &Ator{}
	return context.WithValue(ctx, CtxKeyAtor, a), a
}

// AtorFromContext devolve o Ator do contexto, ou nil se não houver.
func AtorFromContext(ctx context.Context) *Ator {
	a, _ := ctx.Value(CtxKeyAtor).(*Ator)
	return a
}

// Registrar preenche o Ator. Receptor nil é no-op de propósito: a autenticação
// também roda em testes e em montagens que não passam pelo middleware de
// auditoria, e ali não há Ator no contexto.
func (a *Ator) Registrar(userID, email, role string) {
	if a == nil {
		return
	}
	a.UserID, a.Email, a.Role = userID, email, role
}
