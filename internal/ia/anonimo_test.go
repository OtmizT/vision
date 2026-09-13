package ia

import (
	"strings"
	"testing"
)

func TestAnonimizador_IdaEVolta(t *testing.T) {
	a := NovoAnonimizador()

	r := a.Rotular("Cliente", "Constrimax LTDA")
	if r == "Constrimax LTDA" {
		t.Fatal("o nome real saiu sem rótulo")
	}

	texto := "O maior faturamento veio de " + r + "."
	if got := a.Restaurar(texto); got != "O maior faturamento veio de Constrimax LTDA." {
		t.Fatalf("got %q", got)
	}
}

/*
O mesmo cliente precisa receber o mesmo rótulo em toda a conversa. Sem isso a IA
enxerga dois clientes onde há um, e soma errado.
*/
func TestAnonimizador_MesmoNomeMesmoRotulo(t *testing.T) {
	a := NovoAnonimizador()

	primeiro := a.Rotular("Cliente", "Delta Serviços")
	segundo := a.Rotular("Cliente", "Delta Serviços")
	comEspaco := a.Rotular("Cliente", "  Delta Serviços  ")

	if primeiro != segundo || primeiro != comEspaco {
		t.Fatalf("rótulos divergiram: %q, %q, %q", primeiro, segundo, comEspaco)
	}
	if a.Total() != 1 {
		t.Fatalf("criou %d entradas para um cliente só", a.Total())
	}
}

func TestAnonimizador_NomesDiferentesRotulosDiferentes(t *testing.T) {
	a := NovoAnonimizador()
	vistos := map[string]bool{}

	for _, nome := range []string{"Alpha", "Beta", "Gama", "Delta"} {
		r := a.Rotular("Cliente", nome)
		if vistos[r] {
			t.Fatalf("rótulo repetido: %q", r)
		}
		vistos[r] = true
	}
}

// Prefixos distintos não podem colidir: um "Cliente A" e uma "Empresa A"
// coexistem, e restaurar um não pode arrastar o outro.
func TestAnonimizador_PrefixosIndependentes(t *testing.T) {
	a := NovoAnonimizador()
	cli := a.Rotular("Cliente", "Constrimax")
	emp := a.Rotular("Empresa", "Alpha Comércio")

	if cli == emp {
		t.Fatalf("prefixos colidiram: ambos %q", cli)
	}
	got := a.Restaurar(cli + " e " + emp)
	if got != "Constrimax e Alpha Comércio" {
		t.Fatalf("got %q", got)
	}
}

/*
O caso que a ordem de substituição existe para resolver.

Com 27 clientes existem "Cliente A" e "Cliente AA". Substituindo do mais curto
para o mais longo, "Cliente A" casaria DENTRO de "Cliente AA" e produziria
"<nome de A>A" — um nome que não existe, numa resposta sobre dinheiro.
*/
func TestAnonimizador_RotuloLongoNaoEComidoPeloCurto(t *testing.T) {
	a := NovoAnonimizador()

	var primeiro, vigesimoSetimo string
	for i := 1; i <= 27; i++ {
		nome := "Empresa número " + string(rune('a'+i%26)) + string(rune('0'+i/10))
		r := a.Rotular("Cliente", nome)
		if i == 1 {
			primeiro = r
		}
		if i == 27 {
			vigesimoSetimo = r
		}
	}

	if primeiro != "Cliente A" || vigesimoSetimo != "Cliente AA" {
		t.Fatalf("a numeração não chegou ao caso interessante: %q e %q", primeiro, vigesimoSetimo)
	}

	got := a.Restaurar("Comparando " + vigesimoSetimo + " com " + primeiro + ".")
	if strings.Contains(got, "Cliente") {
		t.Fatalf("sobrou rótulo sem restaurar: %q", got)
	}
	// O nome do 27º tem de aparecer inteiro, não o do 1º seguido de "A".
	if !strings.Contains(got, a.paraReal["Cliente AA"]) {
		t.Fatalf("o rótulo longo foi comido pelo curto: %q", got)
	}
}

// "Não informado" já chega assim do banco; rotular isso criaria um cliente
// fantasma que a IA trataria como entidade real.
func TestAnonimizador_NomeVazioNaoVirarRotulo(t *testing.T) {
	a := NovoAnonimizador()
	for _, vazio := range []string{"", "   "} {
		if r := a.Rotular("Cliente", vazio); r != "" {
			t.Fatalf("nome vazio virou %q", r)
		}
	}
	if a.Total() != 0 {
		t.Fatalf("criou %d entradas para nada", a.Total())
	}
}

// Modelo alucina. Um rótulo que não existe no mapa não pode derrubar a resposta
// inteira — é melhor a tela mostrar "Cliente Z" do que não mostrar nada.
func TestAnonimizador_RotuloDesconhecidoNaoQuebra(t *testing.T) {
	a := NovoAnonimizador()
	a.Rotular("Cliente", "Constrimax")

	got := a.Restaurar("Comparei Cliente A com Cliente Z.")
	if !strings.Contains(got, "Constrimax") {
		t.Fatalf("o rótulo conhecido não foi restaurado: %q", got)
	}
	if !strings.Contains(got, "Cliente Z") {
		t.Fatalf("o rótulo inventado deveria passar intacto: %q", got)
	}
}

func TestAnonimizador_SemMapaTextoIntacto(t *testing.T) {
	a := NovoAnonimizador()
	const texto = "Receita de setembro: R$ 6,07 mi."
	if got := a.Restaurar(texto); got != texto {
		t.Fatalf("got %q", got)
	}
}

func TestSufixo(t *testing.T) {
	casos := map[int]string{1: "A", 2: "B", 26: "Z", 27: "AA", 28: "AB", 52: "AZ", 53: "BA", 703: "AAA"}
	for n, esperado := range casos {
		if got := sufixo(n); got != esperado {
			t.Errorf("sufixo(%d) = %q, esperava %q", n, got, esperado)
		}
	}
	// Nunca devolver string vazia: um rótulo vazio viraria "Cliente " e
	// casaria com tudo na restauração.
	if sufixo(0) == "" || sufixo(-1) == "" {
		t.Fatal("sufixo devolveu vazio")
	}
}

/*
A garantia que importa: NENHUM nome real pode sobrar no texto que sai daqui.
Percorre todos os nomes rotulados e confirma que nenhum aparece.
*/
func TestAnonimizador_NenhumNomeRealVazaNaSaida(t *testing.T) {
	a := NovoAnonimizador()
	nomes := []string{"Constrimax LTDA", "Delta Serviços ME", "Alpha Comércio S/A"}

	var partes []string
	for _, n := range nomes {
		partes = append(partes, a.Rotular("Cliente", n))
	}
	enviado := "Top 3: " + strings.Join(partes, ", ") + "."

	for _, n := range nomes {
		if strings.Contains(enviado, n) {
			t.Fatalf("o nome %q foi para o provedor em claro: %q", n, enviado)
		}
	}
}
