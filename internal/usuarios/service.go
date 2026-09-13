package usuarios

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"omie-sync-api/internal/apperror"
)

type Service interface {
	Create(ctx context.Context, grupoID string, req CreateRequest) (*CreateResult, error)
	GetByID(ctx context.Context, id string) (*Usuario, error)
	List(ctx context.Context, params ListParams) ([]*Usuario, int64, error)
	Update(ctx context.Context, id, grupoID string, req UpdateRequest) (*Usuario, error)
	UpdatePassword(ctx context.Context, id string, req UpdatePasswordRequest) error
	Delete(ctx context.Context, id string) error
}

/*
TokenRevoker derruba as sessoes de um usuario.

internal/auth.Repository ja satisfaz a interface — RevokeAllUserTokens existe
desde a fase 3 e nunca foi chamada de lugar nenhum. Sem ela, rebaixar alguem
nao tinha efeito por ate sete dias: o access token ja emitido segue valido, e o
refresh reemite outro com o papel antigo.
*/
type TokenRevoker interface {
	RevokeAllUserTokens(ctx context.Context, usuarioID string) error
}

type service struct {
	repo Repository
	// revoker pode ser nil em montagens de teste.
	revoker TokenRevoker
}

func NewService(repo Repository, revoker TokenRevoker) Service {
	return &service{repo: repo, revoker: revoker}
}

func (s *service) Create(ctx context.Context, grupoID string, req CreateRequest) (*CreateResult, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))

	// Verificar se o usuário já existe pelo e-mail
	existing, err := s.repo.GetByEmail(ctx, email)
	if err == nil && existing != nil {
		// Usuário já existe — verificar se já está neste grupo
		already, err := s.repo.HasGrupoVinculo(ctx, existing.ID, grupoID)
		if err != nil {
			return nil, fmt.Errorf("usuarios.service.Create verificar vínculo: %w", err)
		}
		if already {
			return nil, apperror.Conflict("usuário já pertence a este grupo")
		}
		// Adicionar ao grupo sem recriar (preserva o role solicitado)
		addRole := req.Role
		if addRole == "" {
			addRole = "viewer"
		}
		if err := s.repo.InsertGrupoVinculo(ctx, existing.ID, grupoID, addRole); err != nil {
			if errors.Is(err, ErrMigrationPendente) {
				return nil, apperror.Unprocessable("funcionalidade multi-grupo indisponível: migration 000023 pendente no banco de dados")
			}
			return nil, fmt.Errorf("usuarios.service.Create vincular grupo existente: %w", err)
		}
		return &CreateResult{Usuario: existing, AddedToGroup: true}, nil
	}

	// Usuário novo — validação completa
	if err := validateCreate(req); err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("usuarios.service.Create hash password: %w", err)
	}

	role := req.Role
	if role == "" {
		role = "viewer"
	}

	u, err := s.repo.Insert(ctx, grupoID,
		strings.TrimSpace(req.Nome),
		email,
		string(hash),
		role,
	)
	if err != nil {
		return nil, fmt.Errorf("usuarios.service.Create: %w", err)
	}

	if err := s.repo.InsertGrupoVinculo(ctx, u.ID, grupoID, role); err != nil {
		if errors.Is(err, ErrMigrationPendente) {
			return nil, apperror.Unprocessable("funcionalidade multi-grupo indisponível: migration 000023 pendente no banco de dados")
		}
		return nil, fmt.Errorf("usuarios.service.Create vincular grupo: %w", err)
	}

	return &CreateResult{Usuario: u, AddedToGroup: false}, nil
}

func (s *service) GetByID(ctx context.Context, id string) (*Usuario, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperror.NotFound("usuário não encontrado")
	}
	return u, nil
}

func (s *service) List(ctx context.Context, params ListParams) ([]*Usuario, int64, error) {
	if params.PerPage <= 0 {
		params.PerPage = 50
	}
	if params.Page <= 0 {
		params.Page = 1
	}
	offset := int32((params.Page - 1) * params.PerPage)

	us, err := s.repo.List(ctx, params.GrupoID, int32(params.PerPage), offset)
	if err != nil {
		return nil, 0, fmt.Errorf("usuarios.service.List: %w", err)
	}
	total, err := s.repo.Count(ctx, params.GrupoID)
	if err != nil {
		return nil, 0, fmt.Errorf("usuarios.service.List count: %w", err)
	}
	return us, total, nil
}

func (s *service) Update(ctx context.Context, id, grupoID string, req UpdateRequest) (*Usuario, error) {
	if strings.TrimSpace(req.Nome) == "" {
		return nil, apperror.Unprocessable("nome é obrigatório")
	}
	if req.Role != "" && !rolesValidas[req.Role] {
		return nil, apperror.Unprocessable("role inválida")
	}

	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return nil, apperror.NotFound("usuário não encontrado")
	}

	anterior, err := s.repo.RoleNoGrupo(ctx, id, grupoID)
	if err != nil {
		return nil, fmt.Errorf("usuarios.service.Update ler papel atual: %w", err)
	}

	/*
	 * Papel ausente no corpo preserva o que estava, e nao rebaixa para viewer.
	 *
	 * Um PUT que so quisesse corrigir o nome apagava o papel do usuario em
	 * silencio: 200 OK, resposta com o nome novo, e o administrador do grupo
	 * descobria dias depois que perdeu o proprio acesso administrativo. Nada
	 * no retorno anunciava a troca.
	 */
	role := req.Role
	if role == "" {
		role = anterior
	}
	if role == "" {
		role = "viewer"
	}

	u, err := s.repo.Update(ctx, id, grupoID, strings.TrimSpace(req.Nome), role, req.Ativo)
	if err != nil {
		return nil, fmt.Errorf("usuarios.service.Update: %w", err)
	}

	// Mudanca de papel e desativacao so valem quando a sessao em curso cai.
	if role != anterior || !req.Ativo {
		s.revogarSessoes(ctx, id)
	}
	return u, nil
}

/*
revogarSessoes e best-effort de proposito.

A alteracao ja foi gravada quando chegamos aqui. Devolver erro faria o
administrador repetir um PUT que ja surtiu efeito, e a repeticao tenderia a
falhar no mesmo ponto. O custo de nao revogar e limitado: o access token expira
em 15 minutos, e o refresh e o unico caminho para renova-lo.
*/
func (s *service) revogarSessoes(ctx context.Context, usuarioID string) {
	if s.revoker == nil {
		return
	}
	_ = s.revoker.RevokeAllUserTokens(ctx, usuarioID)
}

func (s *service) UpdatePassword(ctx context.Context, id string, req UpdatePasswordRequest) error {
	if len(req.Password) < 8 {
		return apperror.Unprocessable("password deve ter no mínimo 8 caracteres")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("usuarios.service.UpdatePassword hash: %w", err)
	}

	if err := s.repo.UpdatePassword(ctx, id, string(hash)); err != nil {
		return fmt.Errorf("usuarios.service.UpdatePassword: %w", err)
	}
	// Trocar a senha de alguem sem derrubar as sessoes dele nao tira ninguem de
	// dentro: quem estava logado com a senha antiga continua ate sete dias.
	s.revogarSessoes(ctx, id)
	return nil
}

func (s *service) Delete(ctx context.Context, id string) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return apperror.NotFound("usuário não encontrado")
	}
	if err := s.repo.SoftDelete(ctx, id); err != nil {
		return fmt.Errorf("usuarios.service.Delete: %w", err)
	}
	// Soft delete sem revogar deixava o usuario apagado navegando normalmente.
	s.revogarSessoes(ctx, id)
	return nil
}

func validateCreate(req CreateRequest) error {
	if strings.TrimSpace(req.Nome) == "" {
		return apperror.Unprocessable("nome é obrigatório")
	}
	if strings.TrimSpace(req.Email) == "" {
		return apperror.Unprocessable("email é obrigatório")
	}
	if len(req.Password) < 8 {
		return apperror.Unprocessable("password deve ter no mínimo 8 caracteres")
	}
	if req.Role != "" && !rolesValidas[req.Role] {
		return apperror.Unprocessable("role inválida")
	}
	return nil
}
