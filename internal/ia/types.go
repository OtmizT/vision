package ia

import "time"

// TipoResposta: o que a IA produziu.
type TipoResposta string

const (
	// RespostaTexto é a resposta comum, em Markdown.
	RespostaTexto TipoResposta = "resposta"
	// RespostaGrafico traz também uma especificação de gráfico.
	RespostaGrafico TipoResposta = "grafico"
	// RespostaRecusa é cidadã de primeira classe, e não um erro: "esse dado não
	// está disponível" vale mais que um número plausível e errado.
	RespostaRecusa TipoResposta = "recusa"
)

type PerguntaRequest struct {
	Pergunta string `json:"pergunta"`
	/*
	 * Filtros da tela em que a pessoa está.
	 *
	 * Vão como PADRÃO para as ferramentas, não como dado: "e em agosto?" só faz
	 * sentido se o sistema souber que estamos olhando setembro. A IA pode
	 * sobrescrever qualquer um deles; o que ela não pode é escolher o grupo.
	 */
	Contexto ContextoTela `json:"contexto"`
}

type ContextoTela struct {
	Ano int    `json:"ano"`
	Mes int    `json:"mes"`
	Aba string `json:"aba"`
}

/*
Fonte é o "mostre a conta".

Toda resposta com número carrega qual ferramenta a produziu e com quais filtros.
É o recurso que faz alguém do financeiro confiar no valor — e o que permite
conferir contra a tela quando não bate.
*/
type Fonte struct {
	Ferramenta string         `json:"ferramenta"`
	Filtros    map[string]any `json:"filtros"`
}

/*
SpecGrafico NÃO é config do Chart.js.

É a descrição do que desenhar. O frontend converte aplicando a paleta, as fontes
e as cores de tema da aplicação — o que garante o padrão visual e evita passar
JSON gerado por modelo direto para `new Chart()`.
*/
type SpecGrafico struct {
	Titulo string `json:"titulo"`
	// barra | barra_horizontal | linha | rosca
	Tipo    string         `json:"tipo"`
	Rotulos []string       `json:"rotulos"`
	Series  []SerieGrafico `json:"series"`
	// Formato dos valores: "moeda" ou "numero".
	Formato string `json:"formato"`
}

type SerieGrafico struct {
	Nome    string    `json:"nome"`
	Valores []float64 `json:"valores"`
}

type Resposta struct {
	Tipo    TipoResposta `json:"tipo"`
	Texto   string       `json:"texto"`
	Grafico *SpecGrafico `json:"grafico,omitempty"`
	Fonte   *Fonte       `json:"fonte,omitempty"`
	Tokens  int32        `json:"tokens"`
}

// Mensagem é uma linha do histórico.
type Mensagem struct {
	ID       string       `json:"id"`
	Papel    string       `json:"papel"` // usuario | assistente
	Conteudo string       `json:"conteudo"`
	Grafico  *SpecGrafico `json:"grafico,omitempty"`
	Fonte    *Fonte       `json:"fonte,omitempty"`
	CriadoEm time.Time    `json:"criado_em"`
}
