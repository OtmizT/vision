package ia

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"omie-sync-api/internal/apperror"
	"omie-sync-api/internal/ia_config"
)

/*
maxRodadas limita o vai-e-vem com o modelo.

Cada rodada é uma chamada paga. Um modelo confuso pode pedir a mesma ferramenta
indefinidamente; sem teto, uma pergunta vira uma fatura. Três cobre o caso real
(pedir ferramenta, receber dado, responder) com uma de folga para quando ele
precisa de duas consultas.
*/
const maxRodadas = 4

// maxHistorico é quantas mensagens anteriores vão como contexto. O histórico
// inteiro cresce sem limite e cada mensagem antiga é token pago em toda
// pergunta nova.
const maxHistorico = 10

type Service interface {
	Perguntar(ctx context.Context, grupoID, usuarioID string, req PerguntaRequest) (*Resposta, error)
	Historico(ctx context.Context, grupoID, usuarioID string) ([]Mensagem, error)
	Limpar(ctx context.Context, grupoID, usuarioID string) error
	Disponivel(ctx context.Context, grupoID string) (bool, error)
}

type service struct {
	pool   *pgxpool.Pool
	repo   Repository
	config ia_config.Service
	log    zerolog.Logger
}

func NewService(pool *pgxpool.Pool, repo Repository, config ia_config.Service, log zerolog.Logger) Service {
	return &service{pool: pool, repo: repo, config: config, log: log}
}

/*
Disponivel é o portão, e ele responde às DUAS perguntas: o recurso está ligado
na plataforma E neste grupo?

A tela usa isto para decidir se mostra o botão. O Perguntar reconfere, porque
esconder um botão não é controle de acesso.
*/
func (s *service) Disponivel(ctx context.Context, grupoID string) (bool, error) {
	cfg, err := s.config.ParaUso(ctx)
	if err != nil {
		return false, err
	}
	if !cfg.Ativo || cfg.APIKey == "" {
		return false, nil
	}
	return s.config.AtivaNoGrupo(ctx, grupoID)
}

func (s *service) Historico(ctx context.Context, grupoID, usuarioID string) ([]Mensagem, error) {
	conversaID, err := s.repo.Conversa(ctx, grupoID, usuarioID)
	if err != nil {
		return nil, err
	}
	return s.repo.Historico(ctx, conversaID, 50)
}

func (s *service) Limpar(ctx context.Context, grupoID, usuarioID string) error {
	return s.repo.Limpar(ctx, grupoID, usuarioID)
}

/*
Perguntar é a orquestração inteira:

	portão → teto de tokens → histórico → rodadas com o modelo → grava

O laço de rodadas existe porque function calling é conversa: o modelo pede uma
ferramenta, recebe o resultado e só então responde. Cada volta pode pedir outra.
*/
func (s *service) Perguntar(ctx context.Context, grupoID, usuarioID string, req PerguntaRequest) (*Resposta, error) {
	pergunta := strings.TrimSpace(req.Pergunta)
	if pergunta == "" {
		return nil, apperror.Unprocessable("a pergunta está vazia")
	}
	if len(pergunta) > 2000 {
		return nil, apperror.Unprocessable("a pergunta é longa demais")
	}

	// O portão reconferido no servidor: esconder o botão na tela não impede
	// ninguém de chamar a API direto.
	ok, err := s.Disponivel(ctx, grupoID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, apperror.Forbidden("o assistente não está habilitado para este grupo")
	}

	cfg, err := s.config.ParaUso(ctx)
	if err != nil {
		return nil, err
	}

	// Teto diário, antes de gastar qualquer token.
	usados, err := s.repo.TokensHoje(ctx, usuarioID)
	if err != nil {
		return nil, err
	}
	if usados >= int64(cfg.TetoTokensDia) {
		return nil, apperror.Unprocessable("você atingiu o limite diário de uso do assistente")
	}

	conversaID, err := s.repo.Conversa(ctx, grupoID, usuarioID)
	if err != nil {
		return nil, err
	}

	// A pergunta é gravada ANTES da chamada ao modelo. Se o provedor falhar, a
	// pergunta continua na conversa e a pessoa vê o que perguntou em vez de uma
	// tela que engoliu o texto dela.
	if err := s.repo.Gravar(ctx, conversaID, Mensagem{Papel: "usuario", Conteudo: pergunta}, 0); err != nil {
		return nil, err
	}

	resp, err := s.conversar(ctx, cfg, conversaID, grupoID, req)
	if err != nil {
		return nil, err
	}

	if err := s.repo.Gravar(ctx, conversaID, Mensagem{
		Papel:    "assistente",
		Conteudo: resp.Texto,
		Grafico:  resp.Grafico,
		Fonte:    resp.Fonte,
	}, resp.Tokens); err != nil {
		// A resposta já existe; perdê-la por falha de gravação seria pior que
		// perder o registro no histórico.
		s.log.Error().Err(err).Msg("ia: falha ao gravar resposta no histórico")
	}

	return resp, nil
}

func (s *service) conversar(ctx context.Context, cfg *ia_config.Config, conversaID, grupoID string, req PerguntaRequest) (*Resposta, error) {
	anon := NovoAnonimizador()
	exec := NovoExecutor(s.pool, grupoID, req.Contexto, anon)
	cli := NovoClient(cfg.BaseURL, cfg.APIKey, cfg.Modelo)

	msgs := []MsgChat{{Papel: "system", Texto: SystemPrompt(exec.ctxTela.Ano, exec.ctxTela.Mes)}}

	// Histórico como contexto: é o que faz "e no mês passado?" funcionar.
	anteriores, err := s.repo.Historico(ctx, conversaID, maxHistorico)
	if err == nil {
		for _, m := range anteriores {
			papel := "user"
			if m.Papel == "assistente" {
				papel = "assistant"
			}
			msgs = append(msgs, MsgChat{Papel: papel, Texto: m.Conteudo})
		}
	}

	var tokens int32
	var fonte *Fonte

	for rodada := 0; rodada < maxRodadas; rodada++ {
		ret, err := cli.Completar(ctx, msgs, Catalogo(), cfg.MaxTokens)
		if err != nil {
			return nil, fmt.Errorf("ia.service.conversar: %w", err)
		}
		tokens += ret.Tokens

		// Sem pedido de ferramenta, é a resposta final.
		if len(ret.Mensagem.ToolCalls) == 0 {
			texto, spec := ExtrairGrafico(anon.Restaurar(ret.Mensagem.Texto))
			tipo := RespostaTexto
			if spec != nil {
				tipo = RespostaGrafico
			}
			if texto == "" && spec == nil {
				return nil, apperror.Unprocessable("o assistente não conseguiu formular uma resposta")
			}
			return &Resposta{Tipo: tipo, Texto: texto, Grafico: spec, Fonte: fonte, Tokens: tokens}, nil
		}

		msgs = append(msgs, ret.Mensagem)

		for _, tc := range ret.Mensagem.ToolCalls {
			resultado, f, err := exec.Executar(ctx, tc.Funcao.Nome, tc.Funcao.Argumentos)
			if err != nil {
				/*
				 * Falha de ferramenta volta para o modelo como texto, e não
				 * como erro HTTP.
				 *
				 * Assim ele pode dizer "não consegui consultar isso" com as
				 * palavras dele, em vez de o usuário receber um 500 sem
				 * explicação. A mensagem é genérica de propósito: o erro real
				 * pode conter nome de tabela ou de coluna.
				 */
				s.log.Warn().Err(err).Str("ferramenta", tc.Funcao.Nome).Msg("ia: ferramenta falhou")
				if errors.Is(err, ErrFerramentaDesconhecida) {
					resultado = `{"erro":"essa consulta não existe"}`
				} else {
					resultado = `{"erro":"não foi possível consultar esses dados agora"}`
				}
			} else if f != nil {
				// A última consulta bem-sucedida é a que a tela mostra como
				// fonte do número.
				fonte = f
			}

			msgs = append(msgs, MsgChat{Papel: "tool", ToolCallID: tc.ID, Texto: resultado})
		}
	}

	/*
	 * Estourou as rodadas: o modelo ficou pedindo ferramenta sem concluir.
	 *
	 * Recusa, e não erro: o usuário precisa saber que a pergunta não foi
	 * respondida, e "não consegui" é informação honesta. O custo já foi pago e
	 * está contabilizado em tokens.
	 */
	return &Resposta{
		Tipo:   RespostaRecusa,
		Texto:  "Não consegui chegar a uma resposta para essa pergunta. Tente reformulá-la de forma mais específica.",
		Tokens: tokens,
	}, nil
}
