package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"omie-sync-api/internal/apperror"
)

const refreshTokenDuration = 7 * 24 * time.Hour

type Service interface {
	Login(ctx context.Context, email, password string) (*LoginResponse, error)
	SelectGrupo(ctx context.Context, preAuthToken, grupoID string) (*LoginResponse, error)
	TrocaGrupo(ctx context.Context, userID, grupoID string) (*LoginResponse, error)
	GetGrupos(ctx context.Context, userID string) ([]GrupoInfo, error)
	Logout(ctx context.Context, refreshToken string) error
	Refresh(ctx context.Context, refreshToken string) (*LoginResponse, error)
	Me(ctx context.Context, userID string) (*MeResponse, error)
}

type service struct {
	repo Repository
	jwt  JWTService
}

func NewService(repo Repository, jwt JWTService) Service {
	return &service{repo: repo, jwt: jwt}
}

func (s *service) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
	usuario, err := s.repo.GetUsuarioByEmail(ctx, email)
	if err != nil {
		return nil, apperror.Unauthorized("credenciais inválidas")
	}

	if !usuario.Ativo {
		return nil, apperror.Unauthorized("usuário inativo")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(usuario.Password), []byte(password)); err != nil {
		return nil, apperror.Unauthorized("credenciais inválidas")
	}

	// Buscar grupos via junction table (fallback para grupo_id legado se tabela ainda não existe)
	grupos, _ := s.repo.GetGruposByUsuarioID(ctx, usuario.ID)

	// Múltiplos grupos: exige seleção antes de emitir access token
	if len(grupos) > 1 {
		preAuthToken, err := s.jwt.GeneratePreAuth(usuario.ID, usuario.Email)
		if err != nil {
			return nil, fmt.Errorf("auth.service.Login gerar pre-auth token: %w", err)
		}
		return &LoginResponse{
			NeedsSelect:  true,
			PreAuthToken: preAuthToken,
			Grupos:       grupos,
		}, nil
	}

	// Único grupo ou fallback para grupo_id legado
	grupoID := usuario.GrupoID
	if len(grupos) == 1 {
		grupoID = grupos[0].ID
	}

	return s.issueTokens(ctx, usuario.ID, grupoID, usuario.Email, s.papelNoGrupo(ctx, usuario, grupoID))
}

func (s *service) SelectGrupo(ctx context.Context, preAuthToken, grupoID string) (*LoginResponse, error) {
	claims, err := s.jwt.ValidatePreAuth(preAuthToken)
	if err != nil {
		return nil, apperror.Unauthorized("pre_auth_token inválido ou expirado")
	}

	pertence, err := s.repo.ValidateUsuarioGrupo(ctx, claims.UserID, grupoID)
	if err != nil {
		return nil, fmt.Errorf("auth.service.SelectGrupo validar grupo: %w", err)
	}
	if !pertence {
		return nil, apperror.Forbidden("usuário não pertence a este grupo")
	}

	usuario, err := s.repo.GetUsuarioByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("auth.service.SelectGrupo buscar usuário: %w", err)
	}

	return s.issueTokens(ctx, usuario.ID, grupoID, usuario.Email, s.papelNoGrupo(ctx, usuario, grupoID))
}

func (s *service) TrocaGrupo(ctx context.Context, userID, grupoID string) (*LoginResponse, error) {
	pertence, err := s.repo.ValidateUsuarioGrupo(ctx, userID, grupoID)
	if err != nil {
		return nil, fmt.Errorf("auth.service.TrocaGrupo validar grupo: %w", err)
	}
	if !pertence {
		return nil, apperror.Forbidden("usuário não pertence a este grupo")
	}

	usuario, err := s.repo.GetUsuarioByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("auth.service.TrocaGrupo buscar usuário: %w", err)
	}

	return s.issueTokens(ctx, usuario.ID, grupoID, usuario.Email, s.papelNoGrupo(ctx, usuario, grupoID))
}

/*
papelNoGrupo devolve o papel com que o token e emitido.

Estava copiado em quatro lugares — Login, SelectGrupo, TrocaGrupo e Refresh —
cada um com sua propria queda para usuarios.Role. Quatro copias de uma regra de
autorizacao sao quatro chances de ela divergir, e a divergencia aqui nao da
erro: da acesso a mais ou a menos do que devia.

A queda silenciosa e mantida de proposito, nao esquecida: GetRoleNoGrupo ja tem
tres niveis internos de fallback para bancos com migrations pendentes, e so
devolve erro quando nem usuarios.role foi legivel. Nesse ponto, negar o login
seria a resposta correta — mas mudar isso muda comportamento, e este passo
deliberadamente nao muda nenhum. Fica com a fase que troca a emissao do token.

A regra nova, por contexto, esta em RoleEfetiva (contexto.go), ja testada e
ainda nao ligada: liga-la aqui tiraria do admin global o papel admin_global no
token, e com ele o acesso ao painel de sync — que e exatamente o corte da fase
seguinte.
*/
func (s *service) papelNoGrupo(ctx context.Context, usuario *Usuario, grupoID string) string {
	role, _ := s.repo.GetRoleNoGrupo(ctx, usuario.ID, grupoID)
	if role == "" {
		role = usuario.Role
	}
	return role
}

func (s *service) issueTokens(ctx context.Context, userID, grupoID, email, role string) (*LoginResponse, error) {
	accessToken, err := s.jwt.Generate(userID, grupoID, email, role)
	if err != nil {
		return nil, fmt.Errorf("auth.service.issueTokens gerar access token: %w", err)
	}

	refreshToken, err := generateOpaqueToken()
	if err != nil {
		return nil, fmt.Errorf("auth.service.issueTokens gerar refresh token: %w", err)
	}

	// O grupo ativo viaja com o refresh token: /auth/refresh recebe só o token
	// opaco, sem as claims do access token expirado, e sem isso não teria como
	// saber qual grupo o usuário multi-grupo havia selecionado.
	if _, err := s.repo.InsertRefreshToken(ctx, userID, refreshToken, time.Now().Add(refreshTokenDuration), grupoID); err != nil {
		return nil, fmt.Errorf("auth.service.issueTokens salvar refresh token: %w", err)
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(15 * time.Minute / time.Second),
	}, nil
}

func (s *service) GetGrupos(ctx context.Context, userID string) ([]GrupoInfo, error) {
	grupos, err := s.repo.GetGruposByUsuarioID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("auth.service.GetGrupos: %w", err)
	}
	return grupos, nil
}

func (s *service) Logout(ctx context.Context, refreshToken string) error {
	if err := s.repo.RevokeRefreshToken(ctx, refreshToken); err != nil {
		return fmt.Errorf("auth.service.Logout: %w", err)
	}
	return nil
}

func (s *service) Refresh(ctx context.Context, refreshToken string) (*LoginResponse, error) {
	rt, err := s.repo.GetRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, apperror.Unauthorized("refresh token inválido ou expirado")
	}

	// Rotação obrigatória: revoga o token atual
	if err := s.repo.RevokeRefreshToken(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("auth.service.Refresh revogar token: %w", err)
	}

	usuario, err := s.repo.GetUsuarioByID(ctx, rt.UsuarioID)
	if err != nil {
		return nil, fmt.Errorf("auth.service.Refresh buscar usuário: %w", err)
	}

	if !usuario.Ativo {
		return nil, apperror.Unauthorized("usuário inativo")
	}

	// O grupo vem do refresh token, não de usuarios.grupo_id. Usar o valor do
	// cadastro devolvia o usuário multi-grupo ao grupo padrão a cada renovação
	// silenciosa — a ~15 min do login a tela trocava de grupo sozinha.
	// Fallback para o cadastro cobre tokens emitidos antes da migration 000029.
	grupoIDForRefresh := rt.GrupoID
	if grupoIDForRefresh == "" {
		grupoIDForRefresh = usuario.GrupoID
	}
	roleForRefresh := s.papelNoGrupo(ctx, usuario, grupoIDForRefresh)
	accessToken, err := s.jwt.Generate(usuario.ID, grupoIDForRefresh, usuario.Email, roleForRefresh)
	if err != nil {
		return nil, fmt.Errorf("auth.service.Refresh gerar access token: %w", err)
	}

	newRefreshToken, err := generateOpaqueToken()
	if err != nil {
		return nil, fmt.Errorf("auth.service.Refresh gerar novo refresh token: %w", err)
	}

	// Propaga o grupo para o token rotacionado, senão a próxima renovação o perderia.
	if _, err := s.repo.InsertRefreshToken(ctx, usuario.ID, newRefreshToken, time.Now().Add(refreshTokenDuration), grupoIDForRefresh); err != nil {
		return nil, fmt.Errorf("auth.service.Refresh salvar novo refresh token: %w", err)
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int(15 * time.Minute / time.Second),
	}, nil
}

func (s *service) Me(ctx context.Context, userID string) (*MeResponse, error) {
	usuario, err := s.repo.GetUsuarioByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("auth.service.Me: %w", err)
	}
	return &MeResponse{
		ID:      usuario.ID,
		GrupoID: usuario.GrupoID,
		Nome:    usuario.Nome,
		Email:   usuario.Email,
		Role:    usuario.Role,
	}, nil
}

func generateOpaqueToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generateOpaqueToken: %w", err)
	}
	return hex.EncodeToString(b), nil
}
