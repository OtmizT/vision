package ia

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	sqlcgen "omie-sync-api/sqlc/generated"
)

type Repository interface {
	// Conversa devolve o id da conversa do par, criando se não existir.
	Conversa(ctx context.Context, grupoID, usuarioID string) (string, error)
	Gravar(ctx context.Context, conversaID string, m Mensagem, tokens int32) error
	Historico(ctx context.Context, conversaID string, limite int32) ([]Mensagem, error)
	Limpar(ctx context.Context, grupoID, usuarioID string) error
	TokensHoje(ctx context.Context, usuarioID string) (int64, error)
	Expurgar(ctx context.Context, dias int32) (int64, error)
}

type repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &repository{pool: pool}
}

func (r *repository) Conversa(ctx context.Context, grupoID, usuarioID string) (string, error) {
	q := sqlcgen.New(r.pool)

	gid, err := paraUUID(grupoID)
	if err != nil {
		return "", fmt.Errorf("ia.repository.Conversa grupo: %w", err)
	}
	uid, err := paraUUID(usuarioID)
	if err != nil {
		return "", fmt.Errorf("ia.repository.Conversa usuario: %w", err)
	}

	row, err := q.UpsertIAConversa(ctx, sqlcgen.UpsertIAConversaParams{GrupoID: gid, UsuarioID: uid})
	if err != nil {
		return "", fmt.Errorf("ia.repository.Conversa: %w", err)
	}
	return uuidToStr(row.ID), nil
}

func (r *repository) Gravar(ctx context.Context, conversaID string, m Mensagem, tokens int32) error {
	q := sqlcgen.New(r.pool)

	cid, err := paraUUID(conversaID)
	if err != nil {
		return fmt.Errorf("ia.repository.Gravar conversa: %w", err)
	}

	// nil vira NULL no JSONB; marshalar um ponteiro nil geraria a string
	// "null", que volta como objeto e confunde a leitura.
	var spec, fonte []byte
	if m.Grafico != nil {
		if spec, err = json.Marshal(m.Grafico); err != nil {
			return fmt.Errorf("ia.repository.Gravar spec: %w", err)
		}
	}
	if m.Fonte != nil {
		if fonte, err = json.Marshal(m.Fonte); err != nil {
			return fmt.Errorf("ia.repository.Gravar fonte: %w", err)
		}
	}

	if _, err := q.InsertIAMensagem(ctx, sqlcgen.InsertIAMensagemParams{
		ConversaID: cid,
		Papel:      m.Papel,
		Conteudo:   m.Conteudo,
		Spec:       spec,
		Fonte:      fonte,
		Tokens:     tokens,
	}); err != nil {
		return fmt.Errorf("ia.repository.Gravar: %w", err)
	}
	return nil
}

/*
Historico devolve as mensagens em ordem CRONOLÓGICA.

A query lê decrescente com LIMIT — é o que permite pegar o fim de uma conversa
longa sem ler tudo — e a inversão acontece aqui. Devolver na ordem do banco
faria a tela mostrar a conversa de trás para frente.
*/
func (r *repository) Historico(ctx context.Context, conversaID string, limite int32) ([]Mensagem, error) {
	q := sqlcgen.New(r.pool)

	cid, err := paraUUID(conversaID)
	if err != nil {
		return nil, fmt.Errorf("ia.repository.Historico: %w", err)
	}

	rows, err := q.ListIAMensagens(ctx, sqlcgen.ListIAMensagensParams{ConversaID: cid, Limit: limite})
	if err != nil {
		return nil, fmt.Errorf("ia.repository.Historico: %w", err)
	}

	out := make([]Mensagem, len(rows))
	for i, row := range rows {
		m := Mensagem{
			ID:       uuidToStr(row.ID),
			Papel:    row.Papel,
			Conteudo: row.Conteudo,
			CriadoEm: row.CreatedAt.Time,
		}
		// JSON inválido no banco não derruba a leitura da conversa inteira: a
		// mensagem aparece sem gráfico, que é melhor que não aparecer.
		if len(row.Spec) > 0 {
			var s SpecGrafico
			if json.Unmarshal(row.Spec, &s) == nil {
				m.Grafico = &s
			}
		}
		if len(row.Fonte) > 0 {
			var f Fonte
			if json.Unmarshal(row.Fonte, &f) == nil {
				m.Fonte = &f
			}
		}
		out[len(rows)-1-i] = m
	}
	return out, nil
}

func (r *repository) Limpar(ctx context.Context, grupoID, usuarioID string) error {
	q := sqlcgen.New(r.pool)

	gid, err := paraUUID(grupoID)
	if err != nil {
		return fmt.Errorf("ia.repository.Limpar grupo: %w", err)
	}
	uid, err := paraUUID(usuarioID)
	if err != nil {
		return fmt.Errorf("ia.repository.Limpar usuario: %w", err)
	}

	if err := q.DeleteIAConversa(ctx, sqlcgen.DeleteIAConversaParams{GrupoID: gid, UsuarioID: uid}); err != nil {
		return fmt.Errorf("ia.repository.Limpar: %w", err)
	}
	return nil
}

func (r *repository) TokensHoje(ctx context.Context, usuarioID string) (int64, error) {
	q := sqlcgen.New(r.pool)

	uid, err := paraUUID(usuarioID)
	if err != nil {
		return 0, fmt.Errorf("ia.repository.TokensHoje: %w", err)
	}

	total, err := q.TokensUsadosHoje(ctx, uid)
	if err != nil {
		return 0, fmt.Errorf("ia.repository.TokensHoje: %w", err)
	}
	return total, nil
}

// Expurgar apaga mensagens antigas e as conversas que ficaram sem nenhuma.
// Devolve quantas mensagens saíram.
func (r *repository) Expurgar(ctx context.Context, dias int32) (int64, error) {
	q := sqlcgen.New(r.pool)

	n, err := q.ExpurgarIAMensagens(ctx, dias)
	if err != nil {
		return 0, fmt.Errorf("ia.repository.Expurgar mensagens: %w", err)
	}
	if _, err := q.ExpurgarIAConversasVazias(ctx); err != nil {
		return n, fmt.Errorf("ia.repository.Expurgar conversas: %w", err)
	}
	return n, nil
}

func paraUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	err := u.Scan(s)
	return u, err
}

func uuidToStr(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", u.Bytes[0:4], u.Bytes[4:6], u.Bytes[6:8], u.Bytes[8:10], u.Bytes[10:16])
}
