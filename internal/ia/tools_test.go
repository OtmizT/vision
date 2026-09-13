package ia

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func executorDeTeste(grupoID string, ctxTela ContextoTela) *Executor {
	// pool nil de propósito: estes testes cobrem a montagem dos parâmetros e o
	// despacho, que acontecem antes de qualquer ida ao banco.
	return NovoExecutor(nil, grupoID, ctxTela, NovoAnonimizador())
}

/*
A garantia central do desenho: o grupo vem das claims, e a IA não tem como
mudá-lo.

Não existe argumento de ferramenta que chegue ao GrupoID — este teste falha se
alguém acrescentar um.
*/
func TestExecutor_GrupoVemDasClaims(t *testing.T) {
	const grupo = "11111111-1111-1111-1111-111111111111"
	e := executorDeTeste(grupo, ContextoTela{Ano: 2026, Mes: 9})

	p := e.params(2026, 9)
	if p.GrupoID != grupo {
		t.Fatalf("GrupoID = %q, esperava %q", p.GrupoID, grupo)
	}
}

/*
O modelo tentando escapar do próprio grupo.

Manda argumentos com nomes plausíveis de grupo; nenhum pode surtir efeito,
porque argsFerramenta não os declara.
*/
func TestExecutor_ArgumentoDaIANaoTrocaOGrupo(t *testing.T) {
	const meu = "11111111-1111-1111-1111-111111111111"
	const outro = "22222222-2222-2222-2222-222222222222"

	e := executorDeTeste(meu, ContextoTela{Ano: 2026, Mes: 9})

	tentativas := []string{
		`{"grupo_id":"` + outro + `"}`,
		`{"grupoID":"` + outro + `"}`,
		`{"schema":"grupo_beta"}`,
		`{"ano":2026,"grupo_id":"` + outro + `","empresa_id":"x"}`,
	}

	for _, raw := range tentativas {
		args := e.parseArgs(json.RawMessage(raw))
		p := e.params(args.Ano, args.Mes)
		if p.GrupoID != meu {
			t.Fatalf("com %s o grupo virou %q", raw, p.GrupoID)
		}
	}
}

// Ferramenta que não existe vira erro identificável, para o service transformar
// em recusa em vez de derrubar a requisição.
func TestExecutor_FerramentaDesconhecida(t *testing.T) {
	e := executorDeTeste("g", ContextoTela{})

	_, _, err := e.Executar(context.Background(), "rodar_sql", nil)
	if !errors.Is(err, ErrFerramentaDesconhecida) {
		t.Fatalf("esperava ErrFerramentaDesconhecida, got %v", err)
	}
}

func TestParseArgs_PadraoDaTela(t *testing.T) {
	e := executorDeTeste("g", ContextoTela{Ano: 2026, Mes: 9})

	// Sem argumentos: herda a tela. É o que faz "e no mês passado?" ter sentido.
	p := e.parseArgs(nil)
	if p.Ano != 2026 || p.Mes != 9 {
		t.Fatalf("got ano=%d mes=%d", p.Ano, p.Mes)
	}

	// Com argumentos válidos: a IA manda.
	p = e.parseArgs(json.RawMessage(`{"ano":2025,"mes":3}`))
	if p.Ano != 2025 || p.Mes != 3 {
		t.Fatalf("got ano=%d mes=%d", p.Ano, p.Mes)
	}
}

/*
Alucinação de argumento não pode custar a resposta.

"mes": 13 e JSON quebrado caem no padrão da tela em vez de virar erro — o
usuário recebe uma resposta sobre o mês certo em vez de uma falha.
*/
func TestParseArgs_ValorInvalidoCaiNoPadrao(t *testing.T) {
	e := executorDeTeste("g", ContextoTela{Ano: 2026, Mes: 9})

	casos := []string{
		`{"mes":13}`,
		`{"mes":0}`,
		`{"mes":-1}`,
		`{"ano":1800}`,
		`{"ano":9999}`,
		`{isso não é json`,
		`{"ano":"dois mil e vinte e seis"}`,
	}

	for _, raw := range casos {
		p := e.parseArgs(json.RawMessage(raw))
		if p.Ano != 2026 || p.Mes != 9 {
			t.Errorf("com %s: ano=%d mes=%d, esperava o padrão da tela", raw, p.Ano, p.Mes)
		}
	}
}

// Sem contexto de tela o executor usa a data de hoje: um chat aberto numa rota
// que não tem filtro de período ainda precisa responder alguma coisa.
func TestNovoExecutor_SemContextoUsaHoje(t *testing.T) {
	e := executorDeTeste("g", ContextoTela{})
	agora := time.Now()

	if e.ctxTela.Ano != agora.Year() {
		t.Fatalf("ano = %d, esperava %d", e.ctxTela.Ano, agora.Year())
	}
	if e.ctxTela.Mes != int(agora.Month()) {
		t.Fatalf("mes = %d, esperava %d", e.ctxTela.Mes, int(agora.Month()))
	}
}

/*
O catálogo é o que o modelo vê. Se uma ferramenta aparecer aqui e não existir no
despacho, o modelo a chamará e receberá erro.
*/
func TestCatalogo_TodaFerramentaAnunciadaExiste(t *testing.T) {
	for _, f := range Catalogo() {
		if !Conhece(f.Nome) {
			t.Errorf("catálogo anuncia %q, que o executor não sabe executar", f.Nome)
		}
	}
}

// E o inverso: ferramenta implementada e não anunciada é código morto que o
// modelo nunca alcança.
func TestCatalogo_TodaFerramentaImplementadaEAnunciada(t *testing.T) {
	anunciadas := map[string]bool{}
	for _, f := range Catalogo() {
		anunciadas[f.Nome] = true
	}
	for nome := range (&Executor{}).despacho() {
		if !anunciadas[nome] {
			t.Errorf("%q está implementada mas não aparece no catálogo", nome)
		}
	}
}

func TestCatalogo_Formato(t *testing.T) {
	cat := Catalogo()
	if len(cat) == 0 {
		t.Fatal("catálogo vazio: o modelo não teria o que chamar")
	}

	vistos := map[string]bool{}
	for _, f := range cat {
		if f.Nome == "" || f.Descricao == "" {
			t.Errorf("ferramenta sem nome ou descrição: %+v", f)
		}
		if vistos[f.Nome] {
			t.Errorf("ferramenta duplicada: %q", f.Nome)
		}
		vistos[f.Nome] = true

		// O provedor recusa a chamada inteira se o schema não for um objeto.
		if f.Parametros["type"] != "object" {
			t.Errorf("%s: schema não é object", f.Nome)
		}
		if _, ok := f.Parametros["properties"]; !ok {
			t.Errorf("%s: schema sem properties", f.Nome)
		}
		// required ausente (nil) faz alguns provedores rejeitarem; precisa ser
		// lista, mesmo que vazia.
		if _, ok := f.Parametros["required"].([]string); !ok {
			t.Errorf("%s: required não é lista", f.Nome)
		}
	}
}

/*
Nenhuma ferramenta pode anunciar parâmetro de grupo, schema ou empresa.

É o teste que impede alguém de, com boa intenção, acrescentar "empresa_id" ao
schema e abrir uma porta para a IA escolher de quem são os dados.
*/
func TestCatalogo_NenhumParametroDeTenant(t *testing.T) {
	proibidos := []string{"grupo", "schema", "tenant", "empresa"}

	for _, f := range Catalogo() {
		props, _ := f.Parametros["properties"].(map[string]any)
		for nome := range props {
			for _, p := range proibidos {
				if strings.Contains(strings.ToLower(nome), p) {
					t.Errorf("%s anuncia o parâmetro %q — a IA não pode escolher de quem são os dados", f.Nome, nome)
				}
			}
		}
	}
}

func TestParaJSON_ValorImpossivelNaoVirarVazio(t *testing.T) {
	// Canal não serializa. O modelo precisa receber algo legível, não "".
	got := paraJSON(map[string]any{"x": make(chan int)})
	if got == "" || !strings.Contains(got, "erro") {
		t.Fatalf("got %q", got)
	}
}
