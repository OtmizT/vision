package ia

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"omie-sync-api/internal/dados"
)

/*
As ferramentas que a IA pode chamar.

Este arquivo é a fronteira do produto. A IA não escreve SQL, não vê o banco e
não conhece nome de tabela: ela escolhe uma destas quatro funções e os filtros.
Quem calcula é o MESMO código que desenha o dashboard — e é daí que vem a
propriedade que faz o recurso valer alguma coisa: o chat não CONSEGUE
contradizer a tela.

É também onde o escopo restrito para de depender do system prompt. Não existe
ferramenta para nada além dos dados financeiros do grupo, então não há o que a
IA consulte fora disso, por mais criativa que seja a pergunta.
*/

/*
timeoutFerramenta corta a consulta antes que o servidor corte a resposta.

O WriteTimeout global é 15s (cmd/api/main.go). Se a consulta usasse tudo, a
conexão morreria com o corpo pela metade e o usuário veria um erro sem causa.
Cortando antes, sobra tempo para a chamada ao modelo e para uma mensagem
honesta de "a consulta demorou demais".

Os endpoints analíticos não têm limite próprio: /pivot devolve o ano inteiro.
*/
const timeoutFerramenta = 8 * time.Second

// maxLinhasPivot limita o que vai para o modelo. Um plano de contas real
// facilmente passa de mil combinações categoria×cliente, e cada linha é token
// pago. Acima disso a resposta é truncada e a IA é avisada disso.
const maxLinhasPivot = 150

type Ferramenta struct {
	Nome      string
	Descricao string
	// Parametros é o JSON Schema que o provedor recebe.
	Parametros map[string]any
}

// Catalogo é o que se oferece ao modelo. Ordem estável para o prompt não mudar
// de uma chamada para outra sem motivo.
func Catalogo() []Ferramenta {
	return []Ferramenta{
		{
			Nome: "resumo_financeiro",
			Descricao: "Totais de receita, despesa, resultado e saldo em contas do ano, " +
				"mais a série mês a mês. Use para perguntas sobre desempenho geral, " +
				"comparação entre meses ou evolução ao longo do ano.",
			Parametros: esquema(map[string]any{
				"ano": map[string]any{"type": "integer", "description": "Ano de referência. Padrão: o da tela."},
			}, nil),
		},
		{
			Nome: "abrir_por_categoria_ou_cliente",
			Descricao: "Abre receitas e despesas por categoria e por cliente/fornecedor, " +
				"com os doze meses do ano. Use para 'quais clientes', 'quais categorias', " +
				"rankings e composição.",
			Parametros: esquema(map[string]any{
				"ano": map[string]any{"type": "integer", "description": "Ano de referência. Padrão: o da tela."},
			}, nil),
		},
		{
			Nome: "fluxo_do_mes",
			Descricao: "Lançamentos de um mês, com o que foi recebido, pago, está a vencer " +
				"e está atrasado. Use para perguntas sobre um mês específico, " +
				"inadimplência ou próximos vencimentos.",
			Parametros: esquema(map[string]any{
				"ano": map[string]any{"type": "integer", "description": "Ano. Padrão: o da tela."},
				"mes": map[string]any{"type": "integer", "description": "Mês de 1 a 12. Padrão: o da tela."},
			}, nil),
		},
		{
			Nome: "opcoes_de_filtro",
			Descricao: "Lista as empresas, contas correntes, departamentos e categorias " +
				"disponíveis. Use quando precisar saber o que existe antes de responder.",
			Parametros: esquema(map[string]any{}, nil),
		},
	}
}

func esquema(props map[string]any, obrigatorios []string) map[string]any {
	if obrigatorios == nil {
		obrigatorios = []string{}
	}
	return map[string]any{
		"type":       "object",
		"properties": props,
		"required":   obrigatorios,
	}
}

/*
Executor roda as ferramentas.

grupoID é campo da struct, e não parâmetro de ferramenta, de propósito: é o que
torna impossível a IA pedir dados de outro grupo. Não há argumento para isso
porque não existe o argumento.
*/
type Executor struct {
	pool    *pgxpool.Pool
	grupoID string
	ctxTela ContextoTela
	anon    *Anonimizador
}

func NovoExecutor(pool *pgxpool.Pool, grupoID string, ctxTela ContextoTela, anon *Anonimizador) *Executor {
	if ctxTela.Ano == 0 {
		ctxTela.Ano = time.Now().Year()
	}
	if ctxTela.Mes == 0 {
		ctxTela.Mes = int(time.Now().Month())
	}
	return &Executor{pool: pool, grupoID: grupoID, ctxTela: ctxTela, anon: anon}
}

// ErrFerramentaDesconhecida: o modelo pediu algo que não existe. Vira recusa,
// não pânico.
var ErrFerramentaDesconhecida = fmt.Errorf("ferramenta desconhecida")

/*
Executar roda a ferramenta e devolve o resultado JÁ PSEUDONIMIZADO, pronto para
ir ao provedor, mais a Fonte para a tela mostrar de onde veio o número.
*/
func (e *Executor) Executar(ctx context.Context, nome string, args json.RawMessage) (string, *Fonte, error) {
	fn, ok := e.despacho()[nome]
	if !ok {
		return "", nil, fmt.Errorf("%w: %q", ErrFerramentaDesconhecida, nome)
	}

	ctx, cancel := context.WithTimeout(ctx, timeoutFerramenta)
	defer cancel()

	return fn(ctx, e.parseArgs(args))
}

/*
despacho mapeia nome de ferramenta para implementação.

Tabela e não switch para que Conhece e Executar leiam a MESMA lista. Com duas
listas, uma ferramenta anunciada no catálogo e ausente do switch só apareceria
quando o modelo a chamasse em produção.
*/
func (e *Executor) despacho() map[string]func(context.Context, argsFerramenta) (string, *Fonte, error) {
	return map[string]func(context.Context, argsFerramenta) (string, *Fonte, error){
		"resumo_financeiro":              e.resumoFinanceiro,
		"abrir_por_categoria_ou_cliente": e.abrirPor,
		"fluxo_do_mes":                   e.fluxoDoMes,
		"opcoes_de_filtro":               func(ctx context.Context, _ argsFerramenta) (string, *Fonte, error) { return e.opcoesDeFiltro(ctx) },
	}
}

// Conhece diz se a ferramenta existe, sem executá-la.
func Conhece(nome string) bool {
	_, ok := (&Executor{}).despacho()[nome]
	return ok
}

/*
params monta os parâmetros da consulta.

É o ÚNICO lugar onde o grupo entra, e ele vem do campo do Executor — que foi
preenchido com as claims do token. A IA nunca toca aqui: não há argumento de
ferramenta que chegue a este campo.
*/
func (e *Executor) params(ano, mes int) dados.DashboardParams {
	return dados.DashboardParams{GrupoID: e.grupoID, Ano: ano, Mes: mes}
}

// argsFerramenta são os únicos parâmetros que a IA controla. Note a ausência de
// qualquer campo de grupo ou schema.
type argsFerramenta struct {
	Ano int `json:"ano"`
	Mes int `json:"mes"`
}

/*
parseArgs aplica os padrões da tela e recusa valores fora da faixa.

Argumento inválido do modelo cai no padrão em vez de virar erro: uma alucinação
de "mes": 13 não deve custar a resposta inteira ao usuário.
*/
func (e *Executor) parseArgs(raw json.RawMessage) argsFerramenta {
	p := argsFerramenta{Ano: e.ctxTela.Ano, Mes: e.ctxTela.Mes}
	if len(raw) == 0 {
		return p
	}

	var recebido argsFerramenta
	if err := json.Unmarshal(raw, &recebido); err != nil {
		return p
	}
	if recebido.Ano >= 2000 && recebido.Ano <= 2100 {
		p.Ano = recebido.Ano
	}
	if recebido.Mes >= 1 && recebido.Mes <= 12 {
		p.Mes = recebido.Mes
	}
	return p
}

func (e *Executor) resumoFinanceiro(ctx context.Context, p argsFerramenta) (string, *Fonte, error) {
	resp, err := dados.QueryDashboard(ctx, e.pool, e.params(p.Ano, 0))
	if err != nil {
		return "", nil, fmt.Errorf("ia.tools.resumoFinanceiro: %w", err)
	}

	saida := map[string]any{
		"ano":             p.Ano,
		"receita_total":   resp.Cards.ReceitaTotal,
		"despesa_total":   resp.Cards.DespesaTotal,
		"resultado":       resp.Cards.Resultado,
		"saldo_em_contas": resp.Cards.SaldoContasCorrentes,
		"por_mes":         resp.GraficoMensal,
	}
	return paraJSON(saida), &Fonte{
		Ferramenta: "resumo_financeiro",
		Filtros:    map[string]any{"ano": p.Ano},
	}, nil
}

func (e *Executor) abrirPor(ctx context.Context, p argsFerramenta) (string, *Fonte, error) {
	resp, err := dados.QueryPivot(ctx, e.pool, e.params(p.Ano, 0))
	if err != nil {
		return "", nil, fmt.Errorf("ia.tools.abrirPor: %w", err)
	}

	linhas := resp.Linhas
	truncado := false
	if len(linhas) > maxLinhasPivot {
		linhas = linhas[:maxLinhasPivot]
		truncado = true
	}

	// Os nomes de cliente saem daqui; categorias vão em claro porque são o
	// vocabulário que dá sentido à pergunta.
	simplificadas := make([]map[string]any, 0, len(linhas))
	for _, l := range linhas {
		simplificadas = append(simplificadas, map[string]any{
			"tipo":               l.Tipo,
			"categoria_superior": l.CategoriaSuperior,
			"categoria":          l.CategoriaFinal,
			"cliente":            e.anon.Rotular("Cliente", l.Cliente),
			"total":              l.Total,
			"meses":              l.Meses,
		})
	}

	saida := map[string]any{
		"ano":             p.Ano,
		"linhas":          simplificadas,
		"resultado_total": resp.ResultadoTotal,
	}
	if truncado {
		saida["aviso"] = fmt.Sprintf(
			"Mostrando as %d primeiras de %d linhas. Diga ao usuário que o recorte foi limitado.",
			maxLinhasPivot, len(resp.Linhas))
	}

	return paraJSON(saida), &Fonte{
		Ferramenta: "abrir_por_categoria_ou_cliente",
		Filtros:    map[string]any{"ano": p.Ano},
	}, nil
}

func (e *Executor) fluxoDoMes(ctx context.Context, p argsFerramenta) (string, *Fonte, error) {
	resp, err := dados.QueryFluxoCaixa(ctx, e.pool, e.params(p.Ano, p.Mes))
	if err != nil {
		return "", nil, fmt.Errorf("ia.tools.fluxoDoMes: %w", err)
	}

	// O resumo e os próximos vencimentos bastam para quase toda pergunta; a
	// lista inteira de lançamentos seriam milhares de linhas de token pago.
	vencimentos := make([]map[string]any, 0, len(resp.ProximosVencimentos))
	for _, v := range resp.ProximosVencimentos {
		vencimentos = append(vencimentos, map[string]any{
			"data":      v.Data,
			"descricao": e.anon.Rotular("Cliente", v.Descricao),
			"categoria": v.Categoria,
			"tipo":      v.Tipo,
			"valor":     v.Valor,
			"status":    v.Status,
		})
	}

	saida := map[string]any{
		"ano":                  p.Ano,
		"mes":                  p.Mes,
		"resumo":               resp.Resumo,
		"total_lancamentos":    len(resp.Transacoes),
		"proximos_vencimentos": vencimentos,
	}
	return paraJSON(saida), &Fonte{
		Ferramenta: "fluxo_do_mes",
		Filtros:    map[string]any{"ano": p.Ano, "mes": p.Mes},
	}, nil
}

func (e *Executor) opcoesDeFiltro(ctx context.Context) (string, *Fonte, error) {
	resp, err := dados.QueryFiltros(ctx, e.pool, e.params(e.ctxTela.Ano, 0))
	if err != nil {
		return "", nil, fmt.Errorf("ia.tools.opcoesDeFiltro: %w", err)
	}

	empresas := make([]string, 0, len(resp.Empresas))
	for _, emp := range resp.Empresas {
		empresas = append(empresas, e.anon.Rotular("Empresa", emp.Nome))
	}

	saida := map[string]any{
		"empresas":      empresas,
		"categorias":    resp.Categorias,
		"departamentos": resp.Departamentos,
	}
	return paraJSON(saida), &Fonte{
		Ferramenta: "opcoes_de_filtro",
		Filtros:    map[string]any{},
	}, nil
}

// paraJSON serializa para o modelo. Falha de serialização vira um objeto de
// erro legível em vez de string vazia — o modelo sabe dizer "não consegui".
func paraJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return `{"erro":"não foi possível serializar o resultado da consulta"}`
	}
	return string(b)
}
