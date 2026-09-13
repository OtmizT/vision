package ia

import (
	"encoding/json"
	"regexp"
	"strings"
)

/*
Extração do gráfico da resposta do modelo.

O modelo escreve texto em Markdown e, quando há gráfico, um bloco ```grafico com
JSON dentro. Separar os dois aqui — e não no navegador — tem duas razões:

  - o texto que vai para a tela sai limpo, sem um bloco de JSON cru no meio;
  - a spec é VALIDADA antes de virar gráfico. JSON malformado ou com série
    desalinhada dos rótulos produz gráfico errado, que é pior que gráfico
    nenhum: um gráfico errado parece certo.
*/

var blocoGrafico = regexp.MustCompile("(?s)```grafico\\s*\\n(.*?)```")

// maxRotulos limita o que vira gráfico. Acima disso as legendas se sobrepõem e
// a figura deixa de comunicar — o próprio prompt já pede para agrupar.
const maxRotulos = 12

var tiposValidos = map[string]bool{
	"barra":            true,
	"barra_horizontal": true,
	"linha":            true,
	"rosca":            true,
}

/*
ExtrairGrafico separa o texto da especificação.

Devolve o texto SEM o bloco e a spec, quando ela existe e é válida. Bloco
inválido é descartado em silêncio para o usuário: ele recebe a explicação em
texto, que é melhor que uma mensagem de erro técnica sobre JSON.
*/
func ExtrairGrafico(resposta string) (string, *SpecGrafico) {
	m := blocoGrafico.FindStringSubmatch(resposta)
	if m == nil {
		return strings.TrimSpace(resposta), nil
	}

	texto := strings.TrimSpace(blocoGrafico.ReplaceAllString(resposta, ""))

	var spec SpecGrafico
	if err := json.Unmarshal([]byte(m[1]), &spec); err != nil {
		return texto, nil
	}
	if !specValida(&spec) {
		return texto, nil
	}
	return texto, &spec
}

/*
specValida recusa o que não vira gráfico honesto.

A checagem que mais importa é a última: série com mais ou menos valores que
rótulos. O Chart.js desenha assim mesmo, alinhando pelo índice — e o resultado é
um gráfico em que a barra de março mostra o valor de abril. Ninguém percebe
olhando.
*/
func specValida(s *SpecGrafico) bool {
	if s == nil || !tiposValidos[s.Tipo] {
		return false
	}
	if len(s.Rotulos) == 0 || len(s.Rotulos) > maxRotulos {
		return false
	}
	if len(s.Series) == 0 {
		return false
	}
	for _, serie := range s.Series {
		if len(serie.Valores) != len(s.Rotulos) {
			return false
		}
	}
	// Rosca com várias séries não existe: só a primeira seria desenhada, e o
	// usuário acharia que está vendo todas.
	if s.Tipo == "rosca" && len(s.Series) > 1 {
		return false
	}
	if s.Formato != "moeda" && s.Formato != "numero" {
		s.Formato = "moeda"
	}
	return true
}
