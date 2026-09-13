package usuarios

import (
	"context"
	"errors"
	"testing"

	"omie-sync-api/internal/apperror"
)

type mockRepo struct {
	usuario     *Usuario
	usuarios    []*Usuario
	total       int64
	err         error
	roleNoGrupo string
}

func (m *mockRepo) Insert(_ context.Context, grupoID, nome, email, _, role string) (*Usuario, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &Usuario{ID: "u1", GrupoID: grupoID, Nome: nome, Email: email, Role: role, Ativo: true}, nil
}
func (m *mockRepo) GetByID(_ context.Context, _ string) (*Usuario, error) {
	if m.usuario == nil {
		return nil, errors.New("não encontrado")
	}
	return m.usuario, nil
}
func (m *mockRepo) List(_ context.Context, _ string, _, _ int32) ([]*Usuario, error) {
	return m.usuarios, m.err
}
func (m *mockRepo) Count(_ context.Context, _ string) (int64, error) { return m.total, m.err }
func (m *mockRepo) Update(_ context.Context, id, _, nome, role string, ativo bool) (*Usuario, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &Usuario{ID: id, Nome: nome, Role: role, Ativo: ativo}, nil
}
func (m *mockRepo) GetByEmail(_ context.Context, _ string) (*Usuario, error) {
	return m.usuario, m.err
}
func (m *mockRepo) HasGrupoVinculo(_ context.Context, _, _ string) (bool, error) {
	return false, m.err
}
func (m *mockRepo) RoleNoGrupo(_ context.Context, _, _ string) (string, error) {
	return m.roleNoGrupo, nil
}
func (m *mockRepo) UpdatePassword(_ context.Context, _, _ string) error        { return m.err }
func (m *mockRepo) SoftDelete(_ context.Context, _ string) error               { return m.err }
func (m *mockRepo) InsertGrupoVinculo(_ context.Context, _, _, _ string) error { return m.err }

func activeUser() *Usuario {
	return &Usuario{ID: "u1", GrupoID: "g1", Nome: "João", Email: "j@t.com", Role: "viewer", Ativo: true}
}

func TestService_Create_Success(t *testing.T) {
	svc := NewService(&mockRepo{}, nil)
	result, err := svc.Create(context.Background(), "g1", CreateRequest{
		Nome: "Ana", Email: "ana@t.com", Password: "senha123",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if result.AddedToGroup {
		t.Error("novo usuário não deveria ter AddedToGroup=true")
	}
	if result.Usuario.Role != "viewer" {
		t.Errorf("role default: got %q", result.Usuario.Role)
	}
	if result.Usuario.Email != "ana@t.com" {
		t.Errorf("email: got %q", result.Usuario.Email)
	}
}

func TestService_Create_PasswordHasheado(t *testing.T) {
	// Verifica que password não é armazenado em plain text
	// O mock captura o hash mas não podemos inspecioná-lo diretamente —
	// o teste garante que não retorna erro (bcrypt não falhou)
	svc := NewService(&mockRepo{}, nil)
	_, err := svc.Create(context.Background(), "g1", CreateRequest{
		Nome: "Bob", Email: "b@t.com", Password: "minhasenha",
	})
	if err != nil {
		t.Fatalf("Create com senha válida não deveria falhar: %v", err)
	}
}

func TestService_Create_Validacoes(t *testing.T) {
	svc := NewService(&mockRepo{}, nil)
	cases := []struct {
		req  CreateRequest
		code int
	}{
		{CreateRequest{Nome: "", Email: "e@t.com", Password: "12345678"}, 422},
		{CreateRequest{Nome: "x", Email: "", Password: "12345678"}, 422},
		{CreateRequest{Nome: "x", Email: "e@t.com", Password: "curta"}, 422},
		{CreateRequest{Nome: "x", Email: "e@t.com", Password: "12345678", Role: "invalida"}, 422},
	}
	for _, tc := range cases {
		_, err := svc.Create(context.Background(), "g1", tc.req)
		ae, ok := apperror.IsAppError(err)
		if !ok || ae.Code != tc.code {
			t.Errorf("req %+v: esperava %d, got %v", tc.req, tc.code, err)
		}
	}
}

func TestService_Update_RoleInvalida(t *testing.T) {
	svc := NewService(&mockRepo{usuario: activeUser()}, nil)
	_, err := svc.Update(context.Background(), "u1", "g1", UpdateRequest{Nome: "X", Role: "superadmin"})
	ae, ok := apperror.IsAppError(err)
	if !ok || ae.Code != 422 {
		t.Errorf("esperava 422, got %v", err)
	}
}

func TestService_Update_NotFound(t *testing.T) {
	svc := NewService(&mockRepo{}, nil)
	_, err := svc.Update(context.Background(), "x", "g1", UpdateRequest{Nome: "X", Role: "viewer"})
	ae, ok := apperror.IsAppError(err)
	if !ok || ae.Code != 404 {
		t.Errorf("esperava 404, got %v", err)
	}
}

func TestService_UpdatePassword_CurtaDemais(t *testing.T) {
	svc := NewService(&mockRepo{usuario: activeUser()}, nil)
	err := svc.UpdatePassword(context.Background(), "u1", UpdatePasswordRequest{Password: "abc"})
	ae, ok := apperror.IsAppError(err)
	if !ok || ae.Code != 422 {
		t.Errorf("esperava 422, got %v", err)
	}
}

func TestService_Delete_NotFound(t *testing.T) {
	svc := NewService(&mockRepo{}, nil)
	err := svc.Delete(context.Background(), "x")
	ae, ok := apperror.IsAppError(err)
	if !ok || ae.Code != 404 {
		t.Errorf("esperava 404, got %v", err)
	}
}

func TestService_Delete_Success(t *testing.T) {
	svc := NewService(&mockRepo{usuario: activeUser()}, nil)
	if err := svc.Delete(context.Background(), "u1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

// --- Fase A/B: papel de plataforma não se concede por esta rota ---

// revokerSpy conta as revogações pedidas.
type revokerSpy struct {
	ids []string
	err error
}

func (r *revokerSpy) RevokeAllUserTokens(_ context.Context, id string) error {
	r.ids = append(r.ids, id)
	return r.err
}

/*
O caminho da escalada: estas rotas ficam sob /admin/grupos/{grupoID}/usuarios e
são operadas por admin_grupo. O PUT grava usuarios.role — a coluna global — e o
CASE WHEN de auth.GetRoleNoGrupo faz admin_global vencer qualquer papel de
grupo. Com admin_global aceito aqui, um admin do cliente promovia a si mesmo a
administrador da plataforma inteira.
*/
func TestService_NaoConcedeAdminGlobal(t *testing.T) {
	t.Run("no update", func(t *testing.T) {
		svc := NewService(&mockRepo{usuario: activeUser(), roleNoGrupo: "viewer"}, nil)
		_, err := svc.Update(context.Background(), "u1", "g1", UpdateRequest{Nome: "X", Role: "admin_global"})
		ae, ok := apperror.IsAppError(err)
		if !ok || ae.Code != 422 {
			t.Fatalf("esperava 422, got %v", err)
		}
	})

	t.Run("no create", func(t *testing.T) {
		svc := NewService(&mockRepo{}, nil)
		_, err := svc.Create(context.Background(), "g1", CreateRequest{
			Nome: "X", Email: "x@y.com", Password: "senha12345", Role: "admin_global",
		})
		ae, ok := apperror.IsAppError(err)
		if !ok || ae.Code != 422 {
			t.Fatalf("esperava 422, got %v", err)
		}
	})

	t.Run("os papéis de grupo continuam válidos", func(t *testing.T) {
		for _, role := range []string{"admin_grupo", "viewer"} {
			svc := NewService(&mockRepo{usuario: activeUser(), roleNoGrupo: "viewer"}, nil)
			if _, err := svc.Update(context.Background(), "u1", "g1", UpdateRequest{Nome: "X", Role: role, Ativo: true}); err != nil {
				t.Fatalf("role %q deveria passar: %v", role, err)
			}
		}
	})
}

/*
O PUT sem o campo role rebaixava para viewer em silêncio: 200 OK, resposta com o
nome novo, e o administrador do grupo descobria dias depois que perdeu o próprio
acesso. Nada no retorno anunciava a troca.
*/
func TestService_Update_SemRoleNaoRebaixa(t *testing.T) {
	repo := &mockRepo{usuario: activeUser(), roleNoGrupo: "admin_grupo"}
	svc := NewService(repo, nil)

	u, err := svc.Update(context.Background(), "u1", "g1", UpdateRequest{Nome: "Nome Novo", Ativo: true})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if u.Role != "admin_grupo" {
		t.Fatalf("papel rebaixado em silêncio: got %q, want admin_grupo", u.Role)
	}
}

// Usuário sem vínculo registrado não tem papel anterior de onde partir.
func TestService_Update_SemRoleESemVinculoCaiEmViewer(t *testing.T) {
	svc := NewService(&mockRepo{usuario: activeUser(), roleNoGrupo: ""}, nil)
	u, err := svc.Update(context.Background(), "u1", "g1", UpdateRequest{Nome: "X", Ativo: true})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if u.Role != "viewer" {
		t.Fatalf("got %q, want viewer", u.Role)
	}
}

/*
RevokeAllUserTokens existe desde a fase 3 e nunca foi chamada. Sem ela,
rebaixar, desativar ou apagar alguém não tirava ninguém de dentro: o refresh
token vale sete dias e reemite access tokens com o papel antigo.
*/
func TestService_RevogaSessoes(t *testing.T) {
	t.Run("ao mudar o papel", func(t *testing.T) {
		spy := &revokerSpy{}
		svc := NewService(&mockRepo{usuario: activeUser(), roleNoGrupo: "admin_grupo"}, spy)
		if _, err := svc.Update(context.Background(), "u1", "g1", UpdateRequest{Nome: "X", Role: "viewer", Ativo: true}); err != nil {
			t.Fatalf("Update: %v", err)
		}
		if len(spy.ids) != 1 || spy.ids[0] != "u1" {
			t.Fatalf("sessões não revogadas: %v", spy.ids)
		}
	})

	t.Run("ao desativar", func(t *testing.T) {
		spy := &revokerSpy{}
		svc := NewService(&mockRepo{usuario: activeUser(), roleNoGrupo: "viewer"}, spy)
		if _, err := svc.Update(context.Background(), "u1", "g1", UpdateRequest{Nome: "X", Role: "viewer", Ativo: false}); err != nil {
			t.Fatalf("Update: %v", err)
		}
		if len(spy.ids) != 1 {
			t.Fatalf("usuário desativado seguiria navegando: %v", spy.ids)
		}
	})

	t.Run("ao trocar a senha", func(t *testing.T) {
		spy := &revokerSpy{}
		svc := NewService(&mockRepo{usuario: activeUser()}, spy)
		if err := svc.UpdatePassword(context.Background(), "u1", UpdatePasswordRequest{Password: "senha-nova-123"}); err != nil {
			t.Fatalf("UpdatePassword: %v", err)
		}
		if len(spy.ids) != 1 {
			t.Fatalf("senha trocada sem derrubar quem tinha a antiga: %v", spy.ids)
		}
	})

	t.Run("ao apagar", func(t *testing.T) {
		spy := &revokerSpy{}
		svc := NewService(&mockRepo{usuario: activeUser()}, spy)
		if err := svc.Delete(context.Background(), "u1"); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		if len(spy.ids) != 1 {
			t.Fatalf("usuário apagado seguiria navegando: %v", spy.ids)
		}
	})

	// Editar só o nome não é motivo para derrubar a sessão de ninguém.
	t.Run("edição sem troca de papel não derruba", func(t *testing.T) {
		spy := &revokerSpy{}
		svc := NewService(&mockRepo{usuario: activeUser(), roleNoGrupo: "viewer"}, spy)
		if _, err := svc.Update(context.Background(), "u1", "g1", UpdateRequest{Nome: "Outro Nome", Role: "viewer", Ativo: true}); err != nil {
			t.Fatalf("Update: %v", err)
		}
		if len(spy.ids) != 0 {
			t.Fatalf("revogou sem necessidade: %v", spy.ids)
		}
	})

	// A alteração já foi gravada quando a revogação roda; falhar aqui faria o
	// administrador repetir um PUT que já surtiu efeito.
	t.Run("falha ao revogar não derruba a operação", func(t *testing.T) {
		spy := &revokerSpy{err: errors.New("banco fora do ar")}
		svc := NewService(&mockRepo{usuario: activeUser(), roleNoGrupo: "admin_grupo"}, spy)
		if _, err := svc.Update(context.Background(), "u1", "g1", UpdateRequest{Nome: "X", Role: "viewer", Ativo: true}); err != nil {
			t.Fatalf("Update deveria concluir: %v", err)
		}
	})
}
