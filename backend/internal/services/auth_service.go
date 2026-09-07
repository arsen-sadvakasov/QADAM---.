package services

import (
	"context"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
)

// Ошибки аутентификации, безопасные для возврата клиенту (не раскрывают,
// существует ли пользователь — защита от enumeration-атак).
var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserInactive       = errors.New("user account is inactive")
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
)

// AuthResult — результат успешного login/refresh: пара токенов для клиента.
type AuthResult struct {
	AccessToken  string
	RefreshToken string
	User         *models.User
}

// AuthService реализует бизнес-логику логина, обновления токена и logout
// (раздел 15 спецификации). Зависит от интерфейсов репозиториев, а не от
// конкретной СУБД (Dependency Inversion — раздел 12).
type AuthService struct {
	users         repositories.UserRepository
	refreshTokens repositories.RefreshTokenRepository
	tokens        *TokenService
}

// NewAuthService создаёт AuthService с внедрёнными зависимостями.
func NewAuthService(users repositories.UserRepository, refreshTokens repositories.RefreshTokenRepository, tokens *TokenService) *AuthService {
	return &AuthService{users: users, refreshTokens: refreshTokens, tokens: tokens}
}

// Login проверяет username/пароль и, при успехе, выпускает пару токенов.
// Возвращает ErrInvalidCredentials как для несуществующего пользователя,
// так и для неверного пароля — единообразный ответ не даёт злоумышленнику
// понять, существует ли аккаунт (NFR-4, OWASP).
func (s *AuthService) Login(ctx context.Context, username, password string) (*AuthResult, error) {
	user, err := s.users.FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if !user.IsActive {
		return nil, ErrUserInactive
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return s.issueTokenPair(ctx, user)
}

// Refresh проверяет refresh-токен, отзывает его (rotation) и выпускает
// новую пару токенов. Ротация refresh-токена при каждом обновлении снижает
// риск повторного использования украденного токена.
func (s *AuthService) Refresh(ctx context.Context, plainRefreshToken string) (*AuthResult, error) {
	tokenHash := HashToken(plainRefreshToken)

	userID, err := s.refreshTokens.FindActiveByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrInvalidRefreshToken
		}
		return nil, err
	}

	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}
	if !user.IsActive {
		return nil, ErrUserInactive
	}

	if err := s.refreshTokens.Revoke(ctx, tokenHash); err != nil {
		return nil, err
	}

	return s.issueTokenPair(ctx, user)
}

// Logout отзывает переданный refresh-токен, инвалидируя сессию (раздел 15).
func (s *AuthService) Logout(ctx context.Context, plainRefreshToken string) error {
	tokenHash := HashToken(plainRefreshToken)
	return s.refreshTokens.Revoke(ctx, tokenHash)
}

func (s *AuthService) issueTokenPair(ctx context.Context, user *models.User) (*AuthResult, error) {
	accessToken, err := s.tokens.GenerateAccessToken(user.ID, user.RoleKey)
	if err != nil {
		return nil, err
	}

	plainRefresh, refreshHash, err := s.tokens.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(RefreshTokenTTL)
	if err := s.refreshTokens.Create(ctx, user.ID, refreshHash, expiresAt); err != nil {
		return nil, err
	}

	_ = s.users.UpdateLastLogin(ctx, user.ID)

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: plainRefresh,
		User:         user,
	}, nil
}

// HashPassword — вспомогательная функция для хэширования паролей bcrypt
// (используется при создании пользователей в Phase 3/5 — Admin Panel).
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}
