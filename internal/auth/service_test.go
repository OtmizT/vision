package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"omie-sync-api/internal/apperror"
)

// --- mock repository ---

type mockRepo struct {
	usuario      *Usuario
	refreshToken *RefreshToken
	insertErr    error
	revokeErr    error
}

func (m *mockRepo) GetUsuarioByEmail(_ context.Context, _ string) (*Usuario, error) {
	if m.usuario == nil {
		return nil, errors.New("não encontrado")
	}
	return m.usuario, nil
}

func (m *mockRepo) GetUsuarioByID(_ context.Context, _ string) (*Usuario, error) {
	if m.usuario == nil {
		return nil, errors.New("não encontrado")
	}
	return m.usuario, nil
}

func (m *mockRepo) InsertRefreshToken(_ context.Context, _, token string, exp time.Time, _ string, _ Contexto) (*RefreshToken, error) {
	if m.insertErr != nil {
		return nil, m.insertErr
	}
	return &RefreshToken{Token: token, ExpiresAt: exp}, nil
}

func (m *mockRepo) UpdateSenhaPropria(_ context.Context, _, _ string) error { return nil }

func (m *mockRepo) GetRefreshToken(_ context.Context, token string) (*RefreshToken, error) {
	if m.refreshToken == nil {
		return nil, errors.New("não encontrado")
	}
	return m.refreshToken, nil
}

func (m *mockRepo) RevokeRefreshToken(_ context.Context, _ string) error {
	return m.revokeErr
}

func (m *mockRepo) RevokeAllUserTokens(_ context.Context, _ string) error {
	return m.revokeErr
}

func (m *mockRepo) GetGruposByUsuarioID(_ context.Context, _ string) ([]GrupoInfo, error) {
	if m.usuario == nil {
		return nil, nil
	}
	// retorna um grupo único para simular usuário mono-grupo
	return []GrupoInfo{{ID: m.usuario.GrupoID, Nome: "Grupo Teste", Slug: "grupo-teste", SchemaName: "grupo_teste"}}, nil
}

func (m *mockRepo) ValidateUsuarioGrupo(_ context.Context, _, _ string) (bool, error) {
	return true, nil
}

func (m *mockRepo) GetRoleNoGrupo(_ context.Context, _, _ string) (string, error) {
	if m.usuario != nil {
		return m.usuario.Role, nil
	}
	return "viewer", nil
}

// --- helpers ---

func hashedPassword(plain string) string {
	h, _ := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.MinCost)
	return string(h)
}

func newTestService(repo Repository) Service {
	return NewService(repo, NewJWTService(testSecret))
}

func activeUser() *Usuario {
	return &Usuario{
		ID:       "user-uuid-1",
		GrupoID:  "grupo-uuid-1",
		Nome:     "Test User",
		Email:    "test@example.com",
		Password: hashedPassword("senha123"),
		Role:     "admin_grupo",
		Ativo:    true,
	}
}

// --- testes ---

func TestService_Login_Success(t *testing.T) {
	svc := newTestService(&mockRepo{usuario: activeUser()})

	resp, err := svc.Login(context.Background(), "test@example.com", "senha123")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("access_token vazio")
	}
	if resp.RefreshToken == "" {
		t.Error("refresh_token vazio")
	}
	if resp.ExpiresIn != 900 {
		t.Errorf("expires_in: got %d want 900", resp.ExpiresIn)
	}
}

func TestService_Login_WrongPassword(t *testing.T) {
	svc := newTestService(&mockRepo{usuario: activeUser()})

	_, err := svc.Login(context.Background(), "test@example.com", "errada")
	if err == nil {
		t.Fatal("esperava erro")
	}
	ae, ok := apperror.IsAppError(err)
	if !ok || ae.Code != 401 {
		t.Errorf("esperava AppError 401, got %v", err)
	}
}

func TestService_Login_UserNotFound(t *testing.T) {
	svc := newTestService(&mockRepo{})

	_, err := svc.Login(context.Background(), "nobody@x.com", "qualquer")
	if err == nil {
		t.Fatal("esperava erro")
	}
	ae, ok := apperror.IsAppError(err)
	if !ok || ae.Code != 401 {
		t.Errorf("esperava AppError 401, got %v", err)
	}
}

func TestService_Login_InactiveUser(t *testing.T) {
	u := activeUser()
	u.Ativo = false
	svc := newTestService(&mockRepo{usuario: u})

	_, err := svc.Login(context.Background(), "test@example.com", "senha123")
	if err == nil {
		t.Fatal("esperava erro")
	}
	ae, ok := apperror.IsAppError(err)
	if !ok || ae.Code != 401 {
		t.Errorf("esperava AppError 401, got %v", err)
	}
}

func TestService_Logout_Success(t *testing.T) {
	svc := newTestService(&mockRepo{})

	err := svc.Logout(context.Background(), "any-token")
	if err != nil {
		t.Fatalf("Logout: %v", err)
	}
}

func TestService_Refresh_Success(t *testing.T) {
	rt := &RefreshToken{
		Token:     "valid-refresh-token",
		UsuarioID: "user-uuid-1",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	svc := newTestService(&mockRepo{usuario: activeUser(), refreshToken: rt})

	resp, err := svc.Refresh(context.Background(), "valid-refresh-token")
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Error("tokens vazios no refresh")
	}
	// novo refresh token deve ser diferente do anterior
	if resp.RefreshToken == "valid-refresh-token" {
		t.Error("refresh token não foi rotacionado")
	}
}

func TestService_Refresh_InvalidToken(t *testing.T) {
	svc := newTestService(&mockRepo{})

	_, err := svc.Refresh(context.Background(), "token-inexistente")
	if err == nil {
		t.Fatal("esperava erro")
	}
	ae, ok := apperror.IsAppError(err)
	if !ok || ae.Code != 401 {
		t.Errorf("esperava AppError 401, got %v", err)
	}
}

func TestService_Me_Success(t *testing.T) {
	svc := newTestService(&mockRepo{usuario: activeUser()})

	me, err := svc.Me(context.Background(), "user-uuid-1")
	if err != nil {
		t.Fatalf("Me: %v", err)
	}
	if me.Email != "test@example.com" {
		t.Errorf("email: got %q", me.Email)
	}
	if me.Role != "admin_grupo" {
		t.Errorf("role: got %q", me.Role)
	}
}
