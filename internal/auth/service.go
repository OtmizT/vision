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
	SelectGrupo(ctx context.Context, preAuthToken string, contexto Contexto, grupoID string) (*LoginResponse, error)
	TrocaGrupo(ctx context.Context, userID string, contexto Contexto, grupoID string) (*LoginResponse, error)
	GetGrupos(ctx context.Context, userID string) ([]GrupoInfo, error)
	GetContextos(ctx context.Context, userID string) (*ContextosResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	Refresh(ctx context.Context, refreshToken string) (*LoginResponse, error)
	Me(ctx context.Context, userID string) (*MeResponse, error)
	TrocarSenhaPropria(ctx context.Context, userID string, contexto Contexto, grupoID string, req TrocaSenhaRequest) (*LoginResponse, error)
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

	decisao := DecidirEntrada(usuario.Role, grupos, usuario.GrupoID)

	if decisao.PedirSelecao {
		preAuthToken, err := s.jwt.GeneratePreAuth(usuario.ID, usuario.Email)
		if err != nil {
			return nil, fmt.Errorf("auth.service.Login gerar pre-auth token: %w", err)
		}
		return &LoginResponse{
			NeedsSelect:  true,
			PreAuthToken: preAuthToken,
			Grupos:       grupos,
			// O admin global com um grupo so tambem cai aqui: sem Plataforma na
			// lista ele entraria direto no grupo, e o painel dele ficaria sem
			// caminho de volta que nao fosse deslogar.
			PodePlataforma: usuario.Role == RolePlataforma,
		}, nil
	}

	return s.emitirNoContexto(ctx, usuario, decisao.Contexto, decisao.GrupoID)
}

/*
emitirNoContexto e o unico lugar que decide o papel do token.

Toda entrada — login, selecao, troca de contexto e renovacao — passa por aqui e
por RoleEfetiva, que e pura e testada. Antes cada um desses quatro caminhos
carregava sua propria copia da regra.
*/
func (s *service) emitirNoContexto(ctx context.Context, usuario *Usuario, contexto Contexto, grupoID string) (*LoginResponse, error) {
	var roleNoGrupo string
	temVinculo := true

	if contexto == ContextoGrupo {
		var err error
		temVinculo, err = s.repo.ValidateUsuarioGrupo(ctx, usuario.ID, grupoID)
		if err != nil {
			return nil, fmt.Errorf("auth.service.emitirNoContexto validar vinculo: %w", err)
		}
		roleNoGrupo = s.papelNoGrupo(ctx, usuario, grupoID)
	}

	role, err := RoleEfetiva(contexto, grupoID, usuario.Role, roleNoGrupo, temVinculo)
	if err != nil {
		return nil, err
	}
	return s.issueTokens(ctx, usuario.ID, grupoID, usuario.Email, role, contexto, usuario.SenhaProvisoria)
}

func (s *service) SelectGrupo(ctx context.Context, preAuthToken string, contexto Contexto, grupoID string) (*LoginResponse, error) {
	claims, err := s.jwt.ValidatePreAuth(preAuthToken)
	if err != nil {
		return nil, apperror.Unauthorized("pre_auth_token inválido ou expirado")
	}

	usuario, err := s.repo.GetUsuarioByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("auth.service.SelectGrupo buscar usuário: %w", err)
	}

	return s.emitirNoContexto(ctx, usuario, contexto, grupoID)
}

// TrocaGrupo troca o contexto ativo sem exigir novo login. O nome ficou do tempo
// em que só se trocava de grupo; hoje Plataforma também é um destino.
func (s *service) TrocaGrupo(ctx context.Context, userID string, contexto Contexto, grupoID string) (*LoginResponse, error) {
	usuario, err := s.repo.GetUsuarioByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("auth.service.TrocaGrupo buscar usuário: %w", err)
	}
	return s.emitirNoContexto(ctx, usuario, contexto, grupoID)
}

/*
GetContextos lista para onde a pessoa pode ir.

Plataforma só aparece para quem é admin_global. A tela poderia deduzir do papel,
mas então a lista de destinos passaria a ser calculada em dois lugares — e o
cliente é o mais fácil de alterar dos dois.
*/
func (s *service) GetContextos(ctx context.Context, userID string) (*ContextosResponse, error) {
	usuario, err := s.repo.GetUsuarioByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("auth.service.GetContextos buscar usuário: %w", err)
	}
	grupos, err := s.repo.GetGruposByUsuarioID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("auth.service.GetContextos buscar grupos: %w", err)
	}
	return &ContextosResponse{
		PodePlataforma: usuario.Role == RolePlataforma,
		Grupos:         grupos,
	}, nil
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

func (s *service) issueTokens(ctx context.Context, userID, grupoID, email, role string, contexto Contexto, senhaProvisoria bool) (*LoginResponse, error) {
	accessToken, err := s.jwt.Generate(userID, grupoID, email, role, contexto, senhaProvisoria)
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
	if _, err := s.repo.InsertRefreshToken(ctx, userID, refreshToken, time.Now().Add(refreshTokenDuration), grupoID, contexto); err != nil {
		return nil, fmt.Errorf("auth.service.issueTokens salvar refresh token: %w", err)
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(15 * time.Minute / time.Second),
		Contexto:     contexto,
		GrupoID:      grupoID,
		// A tela precisa saber logo na resposta do login: a troca vem antes de
		// qualquer outra coisa.
		SenhaProvisoria: senhaProvisoria,
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

	/*
	 * O contexto e o grupo vêm do refresh token, não de usuarios.grupo_id. Usar
	 * o valor do cadastro devolvia o usuário multi-grupo ao grupo padrão a cada
	 * renovação silenciosa — a ~15 min do login a tela trocava sozinha.
	 *
	 * Contexto vazio é token anterior à migration 000031, que os revogou todos;
	 * se algum escapar, 'grupo' é a leitura conservadora — nunca promove
	 * ninguém à plataforma por omissão.
	 */
	contexto := rt.Contexto
	if contexto == "" {
		contexto = ContextoGrupo
	}
	grupoIDForRefresh := rt.GrupoID
	if contexto == ContextoGrupo && grupoIDForRefresh == "" {
		grupoIDForRefresh = usuario.GrupoID
	}

	/*
	 * A renovação revalida, em vez de reemitir o que estava.
	 *
	 * emitirNoContexto relê usuarios.role e o vínculo, e RoleEfetiva recusa o
	 * que não for mais permitido. Sem isso, rebaixar alguém — ou tirá-lo de um
	 * grupo — só surtia efeito quando o refresh token expirasse, até sete dias
	 * depois: a renovação reemitia o papel antigo indefinidamente.
	 *
	 * Quem perdeu o acesso recebe 403 aqui e volta para o login, que é onde a
	 * escolha de contexto acontece.
	 */
	return s.emitirNoContexto(ctx, usuario, contexto, grupoIDForRefresh)
}

func (s *service) Me(ctx context.Context, userID string) (*MeResponse, error) {
	usuario, err := s.repo.GetUsuarioByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("auth.service.Me: %w", err)
	}
	return &MeResponse{
		ID:              usuario.ID,
		GrupoID:         usuario.GrupoID,
		Nome:            usuario.Nome,
		Email:           usuario.Email,
		Role:            usuario.Role,
		SenhaProvisoria: usuario.SenhaProvisoria,
	}, nil
}

/*
TrocarSenhaPropria: a pessoa troca a propria senha, provando a atual.

Nao existia. A tela de Perfil chamava o endpoint ADMINISTRATIVO
(/admin/grupos/{id}/usuarios/{id}/password), que exige papel de admin — e por
isso **um viewer nao conseguia trocar a propria senha**: recebia 403 numa tela
feita para ele. Um admin de grupo conseguia, mas pelo caminho errado: sem provar
a senha atual, entao um token roubado bastava para tomar a conta.

A senha atual e exigida sempre, inclusive quando a atual e provisoria — quem
acabou de entrar com ela a conhece, e abrir excecao aqui seria criar um caminho
de troca sem prova.

Devolve tokens novos: a senha mudou, e as sessoes antigas sao revogadas logo
abaixo. Sem isso a pessoa trocaria a senha e seria deslogada em seguida pela
propria troca.
*/
func (s *service) TrocarSenhaPropria(ctx context.Context, userID string, contexto Contexto, grupoID string, req TrocaSenhaRequest) (*LoginResponse, error) {
	if len(req.SenhaNova) < 8 {
		return nil, apperror.Unprocessable("a senha nova deve ter no mínimo 8 caracteres")
	}
	if req.SenhaNova == req.SenhaAtual {
		return nil, apperror.Unprocessable("a senha nova precisa ser diferente da atual")
	}

	usuario, err := s.repo.GetUsuarioByID(ctx, userID)
	if err != nil {
		return nil, apperror.Unauthorized("usuário não encontrado")
	}
	if bcrypt.CompareHashAndPassword([]byte(usuario.Password), []byte(req.SenhaAtual)) != nil {
		return nil, apperror.Unauthorized("senha atual incorreta")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.SenhaNova), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("auth.service.TrocarSenhaPropria hash: %w", err)
	}
	if err := s.repo.UpdateSenhaPropria(ctx, userID, string(hash)); err != nil {
		return nil, fmt.Errorf("auth.service.TrocarSenhaPropria: %w", err)
	}

	// Trocar a senha derruba as outras sessoes — inclusive a de quem tinha a
	// senha antiga, que e o caso que importa quando ela vazou.
	if err := s.repo.RevokeAllUserTokens(ctx, userID); err != nil {
		return nil, fmt.Errorf("auth.service.TrocarSenhaPropria revogar sessões: %w", err)
	}

	usuario.SenhaProvisoria = false
	// O contexto e o grupo vem das claims do token atual: trocar a senha nao
	// e motivo para mudar onde a pessoa esta.
	return s.emitirNoContexto(ctx, usuario, contexto, grupoID)
}

func generateOpaqueToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generateOpaqueToken: %w", err)
	}
	return hex.EncodeToString(b), nil
}
