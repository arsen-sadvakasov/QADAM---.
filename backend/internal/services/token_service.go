package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/qadam/backend/internal/models"
)

// AccessTokenTTL — время жизни access-токена (раздел 15 спецификации: короткоживущий).
const AccessTokenTTL = 15 * time.Minute

// RefreshTokenTTL — время жизни refresh-токена (раздел 15: 7–30 дней).
const RefreshTokenTTL = 30 * 24 * time.Hour

// AccessClaims — payload JWT access-токена.
type AccessClaims struct {
	UserID string         `json:"sub"`
	Role   models.RoleKey `json:"role"`
	jwt.RegisteredClaims
}

// TokenService отвечает за выпуск и проверку JWT access-токенов, а также
// за генерацию непрозрачных refresh-токенов и их хэширование для хранения в БД.
type TokenService struct {
	accessSecret []byte
}

// NewTokenService создаёт TokenService с секретом для подписи access-токенов.
func NewTokenService(accessSecret string) *TokenService {
	return &TokenService{accessSecret: []byte(accessSecret)}
}

// GenerateAccessToken выпускает подписанный JWT access-токен для пользователя.
func (s *TokenService) GenerateAccessToken(userID string, role models.RoleKey) (string, error) {
	now := time.Now()
	claims := AccessClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenTTL)),
			Subject:   userID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.accessSecret)
}

// ParseAccessToken проверяет подпись и срок действия access-токена и
// возвращает его claims.
func (s *TokenService) ParseAccessToken(tokenString string) (*AccessClaims, error) {
	claims := &AccessClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.accessSecret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// GenerateRefreshToken генерирует криптографически случайный refresh-токен
// (то, что уходит клиенту в httpOnly cookie) и его SHA-256 хэш (то, что
// сохраняется в БД — см. раздел 15: возможность отзыва без хранения токена
// в открытом виде).
func (s *TokenService) GenerateRefreshToken() (plainToken string, tokenHash string, err error) {
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", "", err
	}
	plainToken = base64.RawURLEncoding.EncodeToString(raw)
	tokenHash = HashToken(plainToken)
	return plainToken, tokenHash, nil
}

// HashToken вычисляет SHA-256 хэш переданного refresh-токена в hex-виде —
// используется как при сохранении, так и при поиске токена в БД.
func HashToken(plainToken string) string {
	sum := sha256.Sum256([]byte(plainToken))
	return hex.EncodeToString(sum[:])
}
