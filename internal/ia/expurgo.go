package ia

import (
	"context"
	"time"

	"github.com/rs/zerolog"
)

/*
Expurgo do histórico.

As conversas guardam pergunta e resposta com valores de clientes em texto. Prazo
de descarte é exigência de LGPD, não preferência de produto: dado pessoal não
pode ser retido além do necessário, e "necessário" para um chat de consulta é
curto.

Noventa dias cobre o uso real — conferir o que se perguntou no trimestre — sem
acumular indefinidamente.
*/
const DiasRetencao = 90

// intervaloExpurgo: uma vez ao dia basta. O mesmo molde do deletion_job de
// empresas, que roda de hora em hora para um trabalho mais urgente que este.
const intervaloExpurgo = 24 * time.Hour

type Expurgador struct {
	repo Repository
	log  zerolog.Logger
}

func NovoExpurgador(repo Repository, log zerolog.Logger) *Expurgador {
	return &Expurgador{repo: repo, log: log}
}

/*
Iniciar dispara o job. Bloqueia até o contexto ser cancelado, então o chamador
deve usar `go`.

Roda uma vez ao subir, e não só depois do primeiro intervalo: um servidor que
reinicia todo dia nunca chegaria a expurgar nada.
*/
func (e *Expurgador) Iniciar(ctx context.Context) {
	e.log.Info().Int("dias", DiasRetencao).Msg("ia: expurgo de histórico iniciado")

	e.rodar(ctx)

	t := time.NewTicker(intervaloExpurgo)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			e.log.Info().Msg("ia: expurgo de histórico encerrado")
			return
		case <-t.C:
			e.rodar(ctx)
		}
	}
}

func (e *Expurgador) rodar(ctx context.Context) {
	// Timeout próprio: um expurgo travado não pode segurar o ticker nem o
	// encerramento do processo.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	n, err := e.repo.Expurgar(ctx, DiasRetencao)
	if err != nil {
		e.log.Error().Err(err).Msg("ia: falha no expurgo do histórico")
		return
	}
	if n > 0 {
		e.log.Info().Int64("mensagens", n).Msg("ia: histórico expurgado")
	}
}
