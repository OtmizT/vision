package ia

import (
	"strings"
	"testing"
)

const blocoOK = "```grafico\n" + `{
  "titulo": "Receita por mês",
  "tipo": "barra",
  "rotulos": ["Jan","Fev","Mar"],
  "series": [{"nome":"Receita","valores":[100,200,300]}],
  "formato": "moeda"
}` + "\n```"

func TestExtrairGrafico_SeparaTextoDaSpec(t *testing.T) {
	resposta := "A receita cresceu no trimestre.\n\n" + blocoOK + "\n\nMarço foi o melhor mês."

	texto, spec := ExtrairGrafico(resposta)

	if spec == nil {
		t.Fatal("não extraiu a spec")
	}
	if spec.Titulo != "Receita por mês" || spec.Tipo != "barra" {
		t.Fatalf("spec errada: %+v", spec)
	}
	// O bloco não pode sobrar no texto — apareceria como JSON cru na bolha.
	if strings.Contains(texto, "```") || strings.Contains(texto, "rotulos") {
		t.Fatalf("o bloco vazou para o texto: %q", texto)
	}
	if !strings.Contains(texto, "A receita cresceu") || !strings.Contains(texto, "melhor mês") {
		t.Fatalf("perdeu texto: %q", texto)
	}
}

func TestExtrairGrafico_SemBloco(t *testing.T) {
	texto, spec := ExtrairGrafico("  A receita de setembro foi R$ 6,07 mi.  ")
	if spec != nil {
		t.Fatal("inventou uma spec")
	}
	if texto != "A receita de setembro foi R$ 6,07 mi." {
		t.Fatalf("got %q", texto)
	}
}

/*
Bloco quebrado não pode custar a resposta ao usuário.

O modelo erra o JSON com alguma frequência. Quando isso acontece a explicação em
texto ainda vale — melhor perder o gráfico que perder tudo.
*/
func TestExtrairGrafico_JSONQuebradoPreservaOTexto(t *testing.T) {
	resposta := "Segue a comparação.\n\n```grafico\n{isso não é json}\n```"

	texto, spec := ExtrairGrafico(resposta)
	if spec != nil {
		t.Fatal("aceitou JSON quebrado")
	}
	if !strings.Contains(texto, "Segue a comparação") {
		t.Fatalf("perdeu o texto: %q", texto)
	}
	if strings.Contains(texto, "```") {
		t.Fatalf("sobrou o bloco: %q", texto)
	}
}

/*
O defeito que este arquivo existe para impedir.

Série com mais ou menos valores que rótulos: o Chart.js desenha assim mesmo,
alinhando pelo índice, e a barra de março passa a mostrar o valor de abril.
Ninguém percebe olhando — e é um gráfico sobre dinheiro.
*/
func TestSpecValida_SerieDesalinhadaDosRotulos(t *testing.T) {
	casos := []struct {
		nome    string
		rotulos []string
		valores []float64
	}{
		{"valores a mais", []string{"Jan", "Fev"}, []float64{1, 2, 3}},
		{"valores a menos", []string{"Jan", "Fev", "Mar"}, []float64{1, 2}},
		{"sem valores", []string{"Jan"}, []float64{}},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			s := &SpecGrafico{
				Tipo: "barra", Rotulos: c.rotulos, Formato: "moeda",
				Series: []SerieGrafico{{Nome: "x", Valores: c.valores}},
			}
			if specValida(s) {
				t.Fatalf("aceitou %d rótulos com %d valores", len(c.rotulos), len(c.valores))
			}
		})
	}
}

func TestSpecValida_Recusas(t *testing.T) {
	base := func() *SpecGrafico {
		return &SpecGrafico{
			Tipo: "barra", Rotulos: []string{"Jan"}, Formato: "moeda",
			Series: []SerieGrafico{{Nome: "x", Valores: []float64{1}}},
		}
	}

	t.Run("tipo desconhecido", func(t *testing.T) {
		s := base()
		s.Tipo = "pizza3d"
		if specValida(s) {
			t.Fatal("aceitou tipo que o frontend não sabe desenhar")
		}
	})

	t.Run("sem rótulos", func(t *testing.T) {
		s := base()
		s.Rotulos = nil
		if specValida(s) {
			t.Fatal("aceitou gráfico sem eixo")
		}
	})

	t.Run("sem séries", func(t *testing.T) {
		s := base()
		s.Series = nil
		if specValida(s) {
			t.Fatal("aceitou gráfico sem dado")
		}
	})

	// Acima de 12 as legendas se sobrepõem e a figura deixa de comunicar.
	t.Run("rótulos demais", func(t *testing.T) {
		s := base()
		s.Rotulos = make([]string, maxRotulos+1)
		s.Series[0].Valores = make([]float64, maxRotulos+1)
		if specValida(s) {
			t.Fatalf("aceitou %d rótulos", maxRotulos+1)
		}
	})

	// Rosca com duas séries: só a primeira seria desenhada, e o usuário acharia
	// que está vendo as duas.
	t.Run("rosca com múltiplas séries", func(t *testing.T) {
		s := base()
		s.Tipo = "rosca"
		s.Series = append(s.Series, SerieGrafico{Nome: "y", Valores: []float64{2}})
		if specValida(s) {
			t.Fatal("aceitou rosca com mais de uma série")
		}
	})

	t.Run("nil", func(t *testing.T) {
		if specValida(nil) {
			t.Fatal("aceitou nil")
		}
	})
}

// Formato desconhecido não invalida o gráfico — cai em moeda, que é o caso
// dominante num sistema financeiro.
func TestSpecValida_FormatoDesconhecidoCaiEmMoeda(t *testing.T) {
	s := &SpecGrafico{
		Tipo: "linha", Rotulos: []string{"Jan"}, Formato: "percentual",
		Series: []SerieGrafico{{Nome: "x", Valores: []float64{1}}},
	}
	if !specValida(s) {
		t.Fatal("recusou por causa do formato")
	}
	if s.Formato != "moeda" {
		t.Fatalf("formato = %q, esperava moeda", s.Formato)
	}
}

func TestSpecValida_TodosOsTiposAceitos(t *testing.T) {
	for tipo := range tiposValidos {
		s := &SpecGrafico{
			Tipo: tipo, Rotulos: []string{"Jan"}, Formato: "moeda",
			Series: []SerieGrafico{{Nome: "x", Valores: []float64{1}}},
		}
		if !specValida(s) {
			t.Errorf("recusou o tipo %q, que o catálogo anuncia", tipo)
		}
	}
}
