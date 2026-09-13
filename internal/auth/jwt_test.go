package auth

import (
	"strings"
	"testing"
	"time"
)

const testSecret = "test-secret-minimo-32-caracteres-xpto"

func TestJWT_GenerateAndValidate(t *testing.T) {
	svc := NewJWTService(testSecret)

	token, err := svc.Generate("user-1", "grupo-1", "user@test.com", "admin_grupo", ContextoGrupo)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if token == "" {
		t.Fatal("token vazio")
	}

	claims, err := svc.Validate(token)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	if claims.UserID != "user-1" {
		t.Errorf("UserID: got %q want %q", claims.UserID, "user-1")
	}
	if claims.GrupoID != "grupo-1" {
		t.Errorf("GrupoID: got %q want %q", claims.GrupoID, "grupo-1")
	}
	if claims.Email != "user@test.com" {
		t.Errorf("Email: got %q want %q", claims.Email, "user@test.com")
	}
	if claims.Role != "admin_grupo" {
		t.Errorf("Role: got %q want %q", claims.Role, "admin_grupo")
	}
}

func TestJWT_ValidateInvalidToken(t *testing.T) {
	svc := NewJWTService(testSecret)

	_, err := svc.Validate("token.invalido.aqui")
	if err == nil {
		t.Fatal("esperava erro para token inválido")
	}
}

func TestJWT_ValidateWrongSecret(t *testing.T) {
	svc1 := NewJWTService(testSecret)
	svc2 := NewJWTService("outro-secret-completamente-diferente-aqui")

	token, _ := svc1.Generate("u1", "g1", "a@b.com", "viewer", ContextoGrupo)

	_, err := svc2.Validate(token)
	if err == nil {
		t.Fatal("esperava erro para secret diferente")
	}
}

func TestJWT_ValidateWrongAlgorithm(t *testing.T) {
	svc := NewJWTService(testSecret)

	// Tokens com algoritmo none devem ser rejeitados
	parts := []string{"eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0", "e30", ""}
	fakeToken := strings.Join(parts, ".")

	_, err := svc.Validate(fakeToken)
	if err == nil {
		t.Fatal("esperava erro para algoritmo none")
	}
}

func TestJWT_ExpiresIn15Minutes(t *testing.T) {
	svc := NewJWTService(testSecret)

	token, _ := svc.Generate("u1", "g1", "a@b.com", "viewer", ContextoGrupo)
	claims, err := svc.Validate(token)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	expiry := claims.ExpiresAt.Time
	diff := expiry.Sub(time.Now())

	if diff < 14*time.Minute || diff > 16*time.Minute {
		t.Errorf("expiração fora do esperado: %v", diff)
	}
}
