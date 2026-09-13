package ia

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

/*
Cliente do provedor de modelo.

Fala o dialeto da OpenAI (`/chat/completions` com `tools`), que DeepSeek, Groq e
a própria OpenAI implementam. Trocar de provedor é trocar credencial e base_url.

Segue as convenções de internal/omie/client.go — timeout na struct, retry ciente
de cancelamento, erro embrulhado com contexto — mas NÃO copia dois traços dele:

  - o Omie guarda LastMaskedPayload num campo mutável da struct, o que torna o
    cliente inseguro para uso concorrente. Aqui não há estado por chamada.
  - 30s é curto para geração de texto; o padrão aqui é maior.
*/

const (
	timeoutPadrao = 90 * time.Second
	maxTentativas = 3
)

// esperaEntreTentativas é curta de propósito: há um usuário olhando para um
// cursor piscando. Falha de infra do provedor deve virar mensagem de erro
// rápido, não três minutos de espera.
var esperaEntreTentativas = []time.Duration{1 * time.Second, 3 * time.Second}

const baseURLPadrao = "https://api.deepseek.com"

type Client struct {
	http    *http.Client
	baseURL string
	apiKey  string
	modelo  string
}

func NovoClient(baseURL, apiKey, modelo string) *Client {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = baseURLPadrao
	}
	return &Client{
		http:    &http.Client{Timeout: timeoutPadrao},
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		modelo:  modelo,
	}
}

// --- formato da conversa (dialeto OpenAI) ---

type MsgChat struct {
	Papel string `json:"role"`
	// Texto é omitido quando a mensagem só carrega chamadas de ferramenta.
	Texto      string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type ToolCall struct {
	ID     string        `json:"id"`
	Tipo   string        `json:"type"`
	Funcao FuncaoChamada `json:"function"`
}

type FuncaoChamada struct {
	Nome       string          `json:"name"`
	Argumentos json.RawMessage `json:"arguments"`
}

type requisicao struct {
	Modelo    string     `json:"model"`
	Mensagens []MsgChat  `json:"messages"`
	Tools     []toolSpec `json:"tools,omitempty"`
	MaxTokens int32      `json:"max_tokens,omitempty"`
	// Determinístico o bastante para que a mesma pergunta sobre os mesmos
	// números não produza respostas divergentes de uma hora para a outra.
	Temperatura float64 `json:"temperature"`
}

type toolSpec struct {
	Tipo   string     `json:"type"`
	Funcao funcaoSpec `json:"function"`
}

type funcaoSpec struct {
	Nome       string         `json:"name"`
	Descricao  string         `json:"description"`
	Parametros map[string]any `json:"parameters"`
}

type resposta struct {
	Escolhas []struct {
		Mensagem MsgChat `json:"message"`
		Motivo   string  `json:"finish_reason"`
	} `json:"choices"`
	Uso struct {
		TokensPrompt int32 `json:"prompt_tokens"`
		TokensSaida  int32 `json:"completion_tokens"`
		TokensTotal  int32 `json:"total_tokens"`
	} `json:"usage"`
	Erro *struct {
		Mensagem string `json:"message"`
		Tipo     string `json:"type"`
	} `json:"error"`
}

// Retorno é o que o service consome.
type Retorno struct {
	Mensagem MsgChat
	Tokens   int32
	Truncada bool
}

/*
Completar faz uma rodada de conversa.

Devolve a mensagem do modelo — que pode ser texto ou um pedido de ferramenta — e
o consumo de tokens. Quem orquestra as rodadas é o service.
*/
func (c *Client) Completar(ctx context.Context, msgs []MsgChat, ferramentas []Ferramenta, maxTokens int32) (*Retorno, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("ia.client.Completar: credencial não configurada")
	}

	tools := make([]toolSpec, 0, len(ferramentas))
	for _, f := range ferramentas {
		tools = append(tools, toolSpec{
			Tipo:   "function",
			Funcao: funcaoSpec{Nome: f.Nome, Descricao: f.Descricao, Parametros: f.Parametros},
		})
	}

	corpo, err := json.Marshal(requisicao{
		Modelo:      c.modelo,
		Mensagens:   msgs,
		Tools:       tools,
		MaxTokens:   maxTokens,
		Temperatura: 0.2,
	})
	if err != nil {
		return nil, fmt.Errorf("ia.client.Completar montar corpo: %w", err)
	}

	var ultimoErr error
	for tentativa := 0; tentativa < maxTentativas; tentativa++ {
		if tentativa > 0 {
			espera := esperaEntreTentativas[min(tentativa-1, len(esperaEntreTentativas)-1)]
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(espera):
			}
		}

		ret, err, retentar := c.tentar(ctx, corpo)
		if err == nil {
			return ret, nil
		}
		ultimoErr = err
		if !retentar {
			return nil, err
		}
	}
	return nil, fmt.Errorf("ia.client.Completar após %d tentativas: %w", maxTentativas, ultimoErr)
}

// tentar devolve (resultado, erro, vale-a-pena-retentar).
func (c *Client) tentar(ctx context.Context, corpo []byte) (*Retorno, error, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(corpo))
	if err != nil {
		return nil, fmt.Errorf("ia.client montar request: %w", err), false
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	res, err := c.http.Do(req)
	if err != nil {
		// Falha de rede: vale retentar, a menos que quem chamou desistiu.
		return nil, fmt.Errorf("ia.client chamar provedor: %w", err), ctx.Err() == nil
	}
	defer res.Body.Close()

	bruto, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("ia.client ler resposta: %w", err), true
	}

	/*
	 * Classificação do erro HTTP.
	 *
	 * 429 e 5xx são transitórios e valem retentar. 401 e 400 são configuração
	 * ou programação nossa — retentar só atrasaria a mensagem que o usuário
	 * precisa ver.
	 *
	 * O corpo do erro NÃO entra na mensagem: pode ecoar parte do prompt, e o
	 * prompt carrega dados financeiros.
	 */
	if res.StatusCode != http.StatusOK {
		transitorio := res.StatusCode == http.StatusTooManyRequests || res.StatusCode >= 500
		return nil, fmt.Errorf("ia.client: provedor respondeu %d", res.StatusCode), transitorio
	}

	var r resposta
	if err := json.Unmarshal(bruto, &r); err != nil {
		return nil, fmt.Errorf("ia.client decodificar resposta: %w", err), false
	}
	if r.Erro != nil {
		return nil, fmt.Errorf("ia.client: provedor recusou (%s)", r.Erro.Tipo), false
	}
	if len(r.Escolhas) == 0 {
		return nil, fmt.Errorf("ia.client: provedor devolveu resposta vazia"), false
	}

	return &Retorno{
		Mensagem: r.Escolhas[0].Mensagem,
		Tokens:   r.Uso.TokensTotal,
		Truncada: r.Escolhas[0].Motivo == "length",
	}, nil, false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
