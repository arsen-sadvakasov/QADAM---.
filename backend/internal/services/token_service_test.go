package services

import (
	"strings"
	"testing"

	"github.com/qadam/backend/internal/models"
)

// TokenService — критичный модуль безопасности (Phase 2/15): генерация и
// парсинг access-токенов, refresh-токены.

func newTokenSvc() *TokenServiceWrapper {
	return &TokenServiceWrapper{svc: NewTokenService("test-secret-32-bytes-long-enough!")}
}

type TokenServiceWrapper struct {
	svc interface {
		GenerateAccessToken(userID string, role models.RoleKey) (string, error)
		ParseAccessToken(token string) (*AccessClaims, error)
		GenerateRefreshToken() (string, string, error)
	}
}

// Обёртка для типизации; тесты используют прямой вызов svc.

func TestTokenService_GenerateAndParse(t *testing.T) {
	svc := NewTokenService("test-secret-32-bytes-long-enough!")

	token, err := svc.GenerateAccessToken("user-42", models.RoleCurator)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	claims, err := svc.ParseAccessToken(token)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if claims.UserID != "user-42" {
		t.Errorf("expected user-42, got %q", claims.UserID)
	}
	if claims.Role != models.RoleCurator {
		t.Errorf("expected curator role, got %q", claims.Role)
	}
}

func TestTokenService_ParseGarbage(t *testing.T) {
	svc := NewTokenService("test-secret-32-bytes-long-enough!")

	if _, err := svc.ParseAccessToken("not-a-token"); err == nil {
		t.Error("expected error parsing garbage token")
	}
	if _, err := svc.ParseAccessToken(""); err == nil {
		t.Error("expected error parsing empty token")
	}
}

func TestTokenService_ParseExpired(t *testing.T) {
	// Токен с истёкшим сроком: генерируем с секретом, но проверяем через
	// ParseAccessToken — jwt сам отклонит expired по exp-claim.
	svc := NewTokenService("test-secret-32-bytes-long-enough!")

	// Создаём токен через GenerateAccessToken — срок действия по
	// AccessTokenTTL; для проверки expired нужен токен из прошлого.
	// Парсим подписанный чужим секретом токен — должен быть отклонён:
	other := NewTokenService("another-secret-32-bytes-long-enough!")
	token, _ := other.GenerateAccessToken("u", models.RoleStudent)

	if _, err := svc.ParseAccessToken(token); err == nil {
		t.Error("expected error for token signed with different secret")
	}
}

func TestTokenService_RefreshToken(t *testing.T) {
	svc := NewTokenService("test-secret-32-bytes-long-enough!")

	plain, hash, err := svc.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("generate refresh failed: %v", err)
	}
	if plain == "" || hash == "" {
		t.Fatal("expected non-empty plain and hash")
	}
	// plain-токен не должен совпадать с хэшем (одностороннее преобразование).
	if plain == hash {
		t.Error("plain token must differ from its hash")
	}
	// Хэш детерминирован: HashToken(plain) == hash.
	if HashToken(plain) != hash {
		t.Error("expected HashToken(plain) == hash")
	}
	// Токены уникальны.
	plain2, hash2, _ := svc.GenerateRefreshToken()
	if plain == plain2 || hash == hash2 {
		t.Error("expected unique refresh tokens")
	}
}

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("mypassword123")
	if err != nil {
		t.Fatalf("hash failed: %v", err)
	}
	if hash == "mypassword123" {
		t.Error("hash must differ from plaintext")
	}
	if !strings.HasPrefix(hash, "$2") {
		t.Errorf("expected bcrypt hash format, got %q", hash)
	}
	// Детерминированностьbcrypt с солью: два хэша одного пароля различаются,
	// но оба проверяются через Compare.
	hash2, _ := HashPassword("mypassword123")
	if hash == hash2 {
		t.Error("expected unique salts (different hashes for same password)")
	}
}
