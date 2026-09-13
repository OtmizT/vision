package ia

import (
	"fmt"
	"sort"
	"strings"
)

/*
Pseudonimização dos nomes antes de falar com o provedor.

O provedor é externo e, no caso da DeepSeek, treina sobre os dados por padrão e
não publica janela de retenção. Trabalhamos com dados de dezenas de empresas
clientes; o que sai daqui não deve identificá-las.

A troca é de MÃO DUPLA e vive dentro de uma requisição:

	entrada   "Constrimax LTDA"  →  "Cliente A"   (vai para o provedor)
	saída     "Cliente A"        →  "Constrimax LTDA"  (volta para a tela)

O mapa nunca é persistido nem devolvido ao navegador. Valores, datas e
categorias vão como são — sem eles não há análise nenhuma a fazer.

O que isto NÃO resolve, e é importante não confundir: descrição de categoria
continua indo em claro, porque é o vocabulário que dá sentido à pergunta. Se
um dia isso também precisar sair, o caminho é o mesmo desta estrutura.
*/
type Anonimizador struct {
	// real → rótulo. Guarda o nome como veio.
	paraRotulo map[string]string
	// rótulo → real. É por onde a resposta volta a fazer sentido.
	paraReal map[string]string
	// Próximo índice por prefixo ("Cliente", "Empresa").
	proximo map[string]int
}

func NovoAnonimizador() *Anonimizador {
	return &Anonimizador{
		paraRotulo: map[string]string{},
		paraReal:   map[string]string{},
		proximo:    map[string]int{},
	}
}

/*
Rotular devolve o pseudônimo estável de um nome.

Estável é o ponto: o mesmo cliente precisa receber o mesmo rótulo em toda a
conversa, senão a IA vê dois clientes onde há um e a soma sai errada.

Nome vazio volta vazio — "Não informado" já chega assim do banco e rotular isso
criaria um cliente fantasma.
*/
func (a *Anonimizador) Rotular(prefixo, nome string) string {
	nome = strings.TrimSpace(nome)
	if nome == "" {
		return ""
	}
	if r, ok := a.paraRotulo[nome]; ok {
		return r
	}

	a.proximo[prefixo]++
	rotulo := fmt.Sprintf("%s %s", prefixo, sufixo(a.proximo[prefixo]))

	a.paraRotulo[nome] = rotulo
	a.paraReal[rotulo] = nome
	return rotulo
}

/*
Restaurar devolve os nomes reais ao texto que voltou do provedor.

A substituição é feita do rótulo MAIS LONGO para o mais curto. Sem essa ordem,
"Cliente A" casaria dentro de "Cliente AA" e produziria
"<nome real de A>A" — um nome que não existe, numa resposta que fala de dinheiro.

Rótulo que a IA inventou e que não está no mapa fica como está: é melhor a tela
mostrar "Cliente Z" do que a função falhar e a resposta inteira se perder.
*/
func (a *Anonimizador) Restaurar(texto string) string {
	if texto == "" || len(a.paraReal) == 0 {
		return texto
	}

	rotulos := make([]string, 0, len(a.paraReal))
	for r := range a.paraReal {
		rotulos = append(rotulos, r)
	}
	sort.Slice(rotulos, func(i, j int) bool {
		if len(rotulos[i]) != len(rotulos[j]) {
			return len(rotulos[i]) > len(rotulos[j])
		}
		return rotulos[i] < rotulos[j]
	})

	for _, r := range rotulos {
		texto = strings.ReplaceAll(texto, r, a.paraReal[r])
	}
	return texto
}

/*
sufixo numera em letras: A..Z, depois AA, AB…

Letras e não números porque "Cliente 1" convive mal com valores no mesmo texto —
o modelo às vezes confunde o identificador com a quantia. E a sequência precisa
ser infinita: um plano de contas real passa de 26 clientes sem esforço.
*/
func sufixo(n int) string {
	if n <= 0 {
		return "?"
	}
	var sb []byte
	for n > 0 {
		n--
		sb = append([]byte{byte('A' + n%26)}, sb...)
		n /= 26
	}
	return string(sb)
}

// Total é quantos nomes foram trocados. Serve à verificação: se for zero num
// resultado que tinha clientes, a pseudonimização não rodou.
func (a *Anonimizador) Total() int { return len(a.paraReal) }
