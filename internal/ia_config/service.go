package ia_config

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"omie-sync-api/internal/apperror"
)

type Service interface {
	Get(ctx context.Context) (Response, error)
	Update(ctx context.Context, req UpdateRequest, usuarioID string) (Response, error)
	ListGrupos(ctx context.Context) ([]GrupoIA, error)
	SetGrupo(ctx context.Context, grupoID string, ativa bool, usuarioID string) error

	// ParaUso devolve a configuração COM a chave, para o chat chamar o
	// provedor. Não passa por Response de propósito: é o único caminho por onde
	// a credencial sai daqui, e ele não é alcançável por HTTP.
	ParaUso(ctx context.Context) (*Config, error)
	// AtivaNoGrupo é o portão do chat.
	AtivaNoGrupo(ctx context.Context, grupoID string) (bool, error)
}

type service struct {
	repo Repository

	// Cache da configuração, no molde de omie_config: toda mensagem de chat
	// precisa dela, e ir ao banco a cada uma seria uma consulta por requisição
	// para um dado que muda uma vez por mês.
	//
	// O liga/desliga por grupo NÃO é cacheado: desligar um grupo precisa valer
	// na hora, porque é a decisão de parar de mandar dado daquele cliente para
	// fora. Uma consulta a mais por mensagem é barata perto disso.
	mu     sync.RWMutex
	cache  *Config
	valido bool
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

var provedoresValidos = map[string]bool{
	// Todos falam o dialeto da OpenAI, que é o que o cliente HTTP implementa.
	"deepseek": true,
	"openai":   true,
	"groq":     true,
}

func (s *service) Get(ctx context.Context) (Response, error) {
	cfg, err := s.carregar(ctx)
	if err != nil {
		return Response{}, err
	}
	return toResponse(cfg), nil
}

func (s *service) Update(ctx context.Context, req UpdateRequest, usuarioID string) (Response, error) {
	req.Provedor = strings.TrimSpace(strings.ToLower(req.Provedor))
	req.Modelo = strings.TrimSpace(req.Modelo)
	req.BaseURL = strings.TrimSpace(req.BaseURL)
	req.APIKey = strings.TrimSpace(req.APIKey)

	if !provedoresValidos[req.Provedor] {
		return Response{}, apperror.Unprocessable("provedor inválido")
	}
	if req.Modelo == "" {
		return Response{}, apperror.Unprocessable("modelo é obrigatório")
	}
	if req.MaxTokens <= 0 {
		return Response{}, apperror.Unprocessable("max_tokens deve ser maior que zero")
	}
	if req.TetoTokensDia <= 0 {
		return Response{}, apperror.Unprocessable("teto_tokens_dia deve ser maior que zero")
	}

	atual, err := s.repo.Get(ctx)
	if err != nil {
		return Response{}, err
	}

	/*
	 * Resolução da chave, nesta ordem:
	 *   limpar_chave  → apaga
	 *   api_key vazia → preserva a atual
	 *   api_key cheia → substitui
	 *
	 * Preservar no caso vazio é o que evita a tela precisar receber a
	 * credencial em claro só para salvar uma mudança de modelo.
	 */
	chave := atual.APIKey
	switch {
	case req.LimparChave:
		chave = ""
	case req.APIKey != "":
		chave = req.APIKey
	}

	// Ligar sem credencial deixaria o chat aparecer para o usuário e falhar na
	// primeira pergunta. Melhor recusar aqui, onde há uma tela para explicar.
	if req.Ativo && chave == "" {
		return Response{}, apperror.Unprocessable("não é possível ativar sem uma chave de API")
	}

	if err := s.repo.Update(ctx, req, chave, usuarioID); err != nil {
		return Response{}, err
	}
	s.invalidar()

	cfg, err := s.carregar(ctx)
	if err != nil {
		return Response{}, err
	}
	return toResponse(cfg), nil
}

func (s *service) ListGrupos(ctx context.Context) ([]GrupoIA, error) {
	return s.repo.ListGrupos(ctx)
}

func (s *service) SetGrupo(ctx context.Context, grupoID string, ativa bool, usuarioID string) error {
	if strings.TrimSpace(grupoID) == "" {
		return apperror.Unprocessable("grupo_id é obrigatório")
	}

	// Ligar um grupo enquanto o recurso está desligado na plataforma produziria
	// um estado que a tela mostra como ativo e que não funciona.
	if ativa {
		cfg, err := s.carregar(ctx)
		if err != nil {
			return err
		}
		if !cfg.Ativo {
			return apperror.Unprocessable("o assistente está desativado na configuração geral")
		}
	}

	return s.repo.SetGrupo(ctx, grupoID, ativa, usuarioID)
}

func (s *service) ParaUso(ctx context.Context) (*Config, error) {
	return s.carregar(ctx)
}

func (s *service) AtivaNoGrupo(ctx context.Context, grupoID string) (bool, error) {
	return s.repo.AtivaNoGrupo(ctx, grupoID)
}

func (s *service) carregar(ctx context.Context) (*Config, error) {
	s.mu.RLock()
	if s.valido {
		cfg := s.cache
		s.mu.RUnlock()
		return cfg, nil
	}
	s.mu.RUnlock()

	cfg, err := s.repo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("ia_config.service.carregar: %w", err)
	}

	s.mu.Lock()
	s.cache, s.valido = cfg, true
	s.mu.Unlock()
	return cfg, nil
}

func (s *service) invalidar() {
	s.mu.Lock()
	s.cache, s.valido = nil, false
	s.mu.Unlock()
}
