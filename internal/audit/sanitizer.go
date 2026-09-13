package audit

import (
	"encoding/json"
	"net/url"
	"strings"
)

var sensitiveFields = []string{
	"password",
	"app_secret",
	"token",
	"refresh_token",
	"access_token",
	"secret",
}

// SanitizeBody recebe um JSON em string e substitui campos sensíveis por "***REDACTED***".
// Se o body não for JSON válido, retorna o original sem modificação.
func SanitizeBody(body string) string {
	body = strings.TrimSpace(body)
	if body == "" || body[0] != '{' {
		return body
	}

	var m map[string]any
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		return body
	}

	redactMap(m)

	out, err := json.Marshal(m)
	if err != nil {
		return body
	}
	return string(out)
}

func redactMap(m map[string]any) {
	for k, v := range m {
		if isSensitive(k) {
			m[k] = "***REDACTED***"
			continue
		}
		switch val := v.(type) {
		case map[string]any:
			redactMap(val)
		case []any:
			redactSlice(val)
		}
	}
}

func redactSlice(s []any) {
	for _, item := range s {
		if m, ok := item.(map[string]any); ok {
			redactMap(m)
		}
	}
}

func isSensitive(key string) bool {
	lower := strings.ToLower(key)
	for _, f := range sensitiveFields {
		if lower == f {
			return true
		}
	}
	return false
}

/*
SanitizeQuery redige os valores sensíveis da query string.

As rotas SSE aceitam ?token=<JWT> porque o EventSource do navegador não manda
header Authorization. Guardar a query string crua colocava tokens de acesso
inteiros dentro de audit_logs — uma tabela que existe justamente para ser lida
depois, por gente que não precisa de credencial de ninguém.

A chave é preservada e só o valor some: saber que a requisição levava um token
faz parte da trilha.
*/
func SanitizeQuery(raw string) string {
	if raw == "" {
		return raw
	}
	vals, err := url.ParseQuery(raw)
	if err != nil {
		// Query string malformada: não dá para separar chave de valor com
		// segurança, então não se grava nada em vez de gravar o token inteiro.
		return "[omitido: query inválida]"
	}
	for k, vs := range vals {
		if !isSensitive(k) {
			continue
		}
		for i := range vs {
			vs[i] = "***REDACTED***"
		}
	}
	return vals.Encode()
}
