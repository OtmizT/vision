package audit

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSanitizeBody_RedactsSensitiveFields(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		redacted []string
		kept     []string
	}{
		{
			name:     "redacts password",
			input:    `{"email":"user@example.com","password":"secret123"}`,
			redacted: []string{"password"},
			kept:     []string{"email"},
		},
		{
			name:     "redacts app_secret",
			input:    `{"app_key":"abc","app_secret":"supersecret"}`,
			redacted: []string{"app_secret"},
			kept:     []string{"app_key"},
		},
		{
			name:     "redacts token",
			input:    `{"user_id":"123","token":"jwt.token.here"}`,
			redacted: []string{"token"},
			kept:     []string{"user_id"},
		},
		{
			name:     "redacts nested fields",
			input:    `{"user":{"name":"João","password":"abc"}}`,
			redacted: []string{"password"},
			kept:     []string{"name"},
		},
		{
			name:  "preserves non-sensitive body",
			input: `{"nome":"Alpha","slug":"alpha"}`,
			kept:  []string{"nome", "slug"},
		},
		{
			name:  "returns non-json unchanged",
			input: `not json at all`,
			kept:  []string{},
		},
		{
			name:  "returns empty unchanged",
			input: ``,
			kept:  []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := SanitizeBody(tc.input)

			if len(tc.redacted) > 0 {
				var m map[string]any
				if err := json.Unmarshal([]byte(result), &m); err != nil {
					t.Fatalf("resultado não é JSON válido: %v", err)
				}
				for _, field := range tc.redacted {
					val, ok := m[field]
					if !ok {
						// campo pode estar aninhado — checar string
						if !contains(result, "***REDACTED***") {
							t.Errorf("campo %q não foi redactado", field)
						}
						continue
					}
					if val != "***REDACTED***" {
						t.Errorf("campo %q = %v, queria ***REDACTED***", field, val)
					}
				}
			}

			for _, field := range tc.kept {
				if !contains(result, field) {
					t.Errorf("campo %q foi removido mas não deveria", field)
				}
			}
		})
	}
}

func TestSanitizeBody_CaseInsensitive(t *testing.T) {
	inputs := []string{
		`{"Password":"abc"}`,
		`{"PASSWORD":"abc"}`,
		`{"pAsSwOrD":"abc"}`,
	}
	for _, input := range inputs {
		result := SanitizeBody(input)
		var m map[string]any
		if err := json.Unmarshal([]byte(result), &m); err != nil {
			t.Fatalf("resultado não é JSON: %v", err)
		}
		for _, v := range m {
			if v != "***REDACTED***" {
				t.Errorf("input %q: esperava redact, got %v", input, v)
			}
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}

func TestSanitizeQuery(t *testing.T) {
	casos := []struct {
		nome      string
		raw       string
		contem    []string
		naoContem []string
	}{
		{"vazia", "", nil, nil},
		{"sem nada sensível", "page=2&per_page=50", []string{"page=2", "per_page=50"}, nil},
		{"token do SSE some, a chave fica", "token=abc.def.ghi", []string{"token=", "REDACTED"}, []string{"abc.def.ghi"}},
		{"redige só o valor sensível", "aba=jobs&access_token=xyz", []string{"aba=jobs"}, []string{"xyz"}},
		{"chave repetida some inteira", "token=um&token=dois", []string{"REDACTED"}, []string{"um", "dois"}},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			got := SanitizeQuery(c.raw)
			for _, s := range c.contem {
				if !strings.Contains(got, s) {
					t.Errorf("esperava %q em %q", s, got)
				}
			}
			for _, s := range c.naoContem {
				if strings.Contains(got, s) {
					t.Errorf("não esperava %q em %q", s, got)
				}
			}
		})
	}
}

// Query malformada não dá para separar com segurança: some inteira, em vez de
// ir para o banco com um token dentro.
func TestSanitizeQuery_MalformadaNaoVazaNada(t *testing.T) {
	got := SanitizeQuery("token=%ZZ")
	if strings.Contains(got, "%ZZ") || strings.Contains(got, "token=%") {
		t.Fatalf("got %q", got)
	}
}

/*
A credencial do assistente de IA.

A comparação de isSensitive é por IGUALDADE, não por substring: `api_key` não
casa com `secret` nem com `token`. Sem o nome exato na lista, o PUT que cadastra
a chave gravaria ela em claro em audit_logs — uma tabela feita para ser lida
depois, por gente que não precisa de credencial nenhuma.
*/
func TestSanitizeBody_ChaveDaIA(t *testing.T) {
	corpo := `{"provedor":"deepseek","modelo":"deepseek-v4-flash","api_key":"sk-abc123secreta"}`
	got := SanitizeBody(corpo)

	if strings.Contains(got, "sk-abc123secreta") {
		t.Fatalf("a chave foi para o log em claro: %s", got)
	}
	if !strings.Contains(got, "REDACTED") {
		t.Fatalf("não redigiu: %s", got)
	}
	// O resto do corpo precisa sobreviver — a trilha existe para dizer o que
	// mudou.
	if !strings.Contains(got, "deepseek-v4-flash") {
		t.Fatalf("redigiu demais: %s", got)
	}
}

func TestSanitizeQuery_ChaveDaIA(t *testing.T) {
	if got := SanitizeQuery("api_key=sk-abc123"); strings.Contains(got, "sk-abc123") {
		t.Fatalf("a chave vazou pela query string: %q", got)
	}
}
