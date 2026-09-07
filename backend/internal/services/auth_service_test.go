package services

import (
	"context"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
)

// fakeUserRepository — тестовая заглушка UserRepository, работающая с картой
// в памяти вместо PostgreSQL (изоляция бизнес-логики от СУБД в тестах).
type fakeUserRepository struct {
	byUsername map[string]*models.User
	byID       map[string]*models.User
}

func newFakeUserRepository() *fakeUserRepository {
	return &fakeUserRepository{
		byUsername: make(map[string]*models.User),
		byID:       make(map[string]*models.User),
	}
}

func (f *fakeUserRepository) add(u *models.User) {
	f.byUsername[u.Username] = u
	f.byID[u.ID] = u
}

func (f *fakeUserRepository) FindByUsername(_ context.Context, username string) (*models.User, error) {
	u, ok := f.byUsername[username]
	if !ok {
		return nil, repositories.ErrNotFound
	}
	return u, nil
}

func (f *fakeUserRepository) FindByID(_ context.Context, id string) (*models.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return nil, repositories.ErrNotFound
	}
	return u, nil
}

func (f *fakeUserRepository) UpdateLastLogin(_ context.Context, userID string) error {
	if u, ok := f.byID[userID]; ok {
		now := time.Now()
		u.LastLoginAt = &now
	}
	return nil
}

// fakeRefreshTokenRepository — тестовая заглушка RefreshTokenRepository.
type fakeRefreshTokenRepository struct {
	tokens map[string]refreshTokenRecord
}

type refreshTokenRecord struct {
	userID    string
	expiresAt time.Time
	revoked   bool
}

func newFakeRefreshTokenRepository() *fakeRefreshTokenRepository {
	return &fakeRefreshTokenRepository{tokens: make(map[string]refreshTokenRecord)}
}

func (f *fakeRefreshTokenRepository) Create(_ context.Context, userID, tokenHash string, expiresAt time.Time) error {
	f.tokens[tokenHash] = refreshTokenRecord{userID: userID, expiresAt: expiresAt}
	return nil
}

func (f *fakeRefreshTokenRepository) FindActiveByHash(_ context.Context, tokenHash string) (string, error) {
	rec, ok := f.tokens[tokenHash]
	if !ok || rec.revoked || time.Now().After(rec.expiresAt) {
		return "", repositories.ErrNotFound
	}
	return rec.userID, nil
}

func (f *fakeRefreshTokenRepository) Revoke(_ context.Context, tokenHash string) error {
	rec, ok := f.tokens[tokenHash]
	if !ok {
		return nil
	}
	rec.revoked = true
	f.tokens[tokenHash] = rec
	return nil
}

func (f *fakeRefreshTokenRepository) RevokeAllForUser(_ context.Context, userID string) error {
	for hash, rec := range f.tokens {
		if rec.userID == userID {
			rec.revoked = true
			f.tokens[hash] = rec
		}
	}
	return nil
}

func newTestAuthService() (*AuthService, *fakeUserRepository, *fakeRefreshTokenRepository) {
	users := newFakeUserRepository()
	refreshTokens := newFakeRefreshTokenRepository()
	tokens := NewTokenService("test-access-secret")
	return NewAuthService(users, refreshTokens, tokens), users, refreshTokens
}

func mustHashPassword(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	return string(hash)
}

func TestAuthService_Login_Success(t *testing.T) {
	auth, users, _ := newTestAuthService()
	users.add(&models.User{
		ID:           "user-1",
		Username:     "student1",
		PasswordHash: mustHashPassword(t, "correct-password"),
		FullName:     "Test Student",
		RoleKey:      models.RoleStudent,
		IsActive:     true,
	})

	result, err := auth.Login(context.Background(), "student1", "correct-password")
	if err != nil {
		t.Fatalf("expected successful login, got error: %v", err)
	}
	if result.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if result.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
	if result.User.ID != "user-1" {
		t.Errorf("expected user ID user-1, got %s", result.User.ID)
	}
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	auth, users, _ := newTestAuthService()
	users.add(&models.User{
		ID:           "user-1",
		Username:     "student1",
		PasswordHash: mustHashPassword(t, "correct-password"),
		RoleKey:      models.RoleStudent,
		IsActive:     true,
	})

	_, err := auth.Login(context.Background(), "student1", "wrong-password")
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthService_Login_UnknownUsername(t *testing.T) {
	auth, _, _ := newTestAuthService()

	_, err := auth.Login(context.Background(), "does-not-exist", "whatever")
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials for unknown username, got %v", err)
	}
}

func TestAuthService_Login_InactiveUser(t *testing.T) {
	auth, users, _ := newTestAuthService()
	users.add(&models.User{
		ID:           "user-1",
		Username:     "student1",
		PasswordHash: mustHashPassword(t, "correct-password"),
		RoleKey:      models.RoleStudent,
		IsActive:     false,
	})

	_, err := auth.Login(context.Background(), "student1", "correct-password")
	if err != ErrUserInactive {
		t.Errorf("expected ErrUserInactive, got %v", err)
	}
}

func TestAuthService_Refresh_RotatesToken(t *testing.T) {
	auth, users, refreshTokens := newTestAuthService()
	users.add(&models.User{
		ID:           "user-1",
		Username:     "student1",
		PasswordHash: mustHashPassword(t, "correct-password"),
		RoleKey:      models.RoleStudent,
		IsActive:     true,
	})

	loginResult, err := auth.Login(context.Background(), "student1", "correct-password")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	refreshResult, err := auth.Refresh(context.Background(), loginResult.RefreshToken)
	if err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	if refreshResult.RefreshToken == loginResult.RefreshToken {
		t.Error("expected refresh token rotation, got same token")
	}

	// Старый refresh-токен должен быть отозван и больше не пригоден для повторного использования.
	_, err = auth.Refresh(context.Background(), loginResult.RefreshToken)
	if err != ErrInvalidRefreshToken {
		t.Errorf("expected ErrInvalidRefreshToken for reused old token, got %v", err)
	}

	// Проверяем, что новый токен работает.
	oldHash := HashToken(loginResult.RefreshToken)
	if rec, ok := refreshTokens.tokens[oldHash]; !ok || !rec.revoked {
		t.Error("expected old refresh token to be marked as revoked")
	}
}

func TestAuthService_Refresh_InvalidToken(t *testing.T) {
	auth, _, _ := newTestAuthService()

	_, err := auth.Refresh(context.Background(), "not-a-real-token")
	if err != ErrInvalidRefreshToken {
		t.Errorf("expected ErrInvalidRefreshToken, got %v", err)
	}
}

func TestAuthService_Logout_RevokesToken(t *testing.T) {
	auth, users, _ := newTestAuthService()
	users.add(&models.User{
		ID:           "user-1",
		Username:     "student1",
		PasswordHash: mustHashPassword(t, "correct-password"),
		RoleKey:      models.RoleStudent,
		IsActive:     true,
	})

	loginResult, err := auth.Login(context.Background(), "student1", "correct-password")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	if err := auth.Logout(context.Background(), loginResult.RefreshToken); err != nil {
		t.Fatalf("logout failed: %v", err)
	}

	_, err = auth.Refresh(context.Background(), loginResult.RefreshToken)
	if err != ErrInvalidRefreshToken {
		t.Errorf("expected ErrInvalidRefreshToken after logout, got %v", err)
	}
}
