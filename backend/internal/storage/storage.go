// Package storage предоставляет абстракцию файлового хранилища (раздел 14
// спецификации): backend работает через интерфейс FileStorage, не завязываясь
// на конкретного провайдера (MinIO / Cloudflare R2 / AWS S3 / локальная ФС).
package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ErrNotFound возвращается, когда объект отсутствует в хранилище.
var ErrNotFound = errors.New("storage object not found")

// ErrNotSupported возвращается, когда реализация не поддерживает операцию.
var ErrNotSupported = errors.New("operation not supported by this storage backend")

// FileStorage — абстракция объектного хранилища учебных материалов.
// Реализации: MinIO (S3-совместимое, продакшен) и локальная файловая система
// (разработка без Docker).
type FileStorage interface {
	// Upload сохраняет содержимое файла под заданным ключом.
	Upload(ctx context.Context, key string, r io.Reader, contentType string) error

	// Open возвращает поток чтения объекта по ключу.
	Open(ctx context.Context, key string) (io.ReadSeekCloser, error)

	// Delete удаляет объект по ключу.
	Delete(ctx context.Context, key string) error

	// PresignedGetURL возвращает временную ссылку на скачивание объекта.
	// Реализации, не поддерживающие pre-signed URL, возвращают ErrNotSupported.
	PresignedGetURL(ctx context.Context, key string, expires time.Duration) (string, error)
}

// NewKey генерирует безопасный ключ объекта вида
// "materials/<subject_id>/<random>.<ext>" — непредсказуемое имя исключает
// коллизии и перебор, а префикс по предмету упрощает навигацию/очистку.
func NewKey(subjectID, originalFileName string) (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	ext := strings.ToLower(filepath.Ext(originalFileName))
	return "materials/" + subjectID + "/" + hex.EncodeToString(buf) + ext, nil
}

// LocalFileStorage — реализация FileStorage поверх локальной файловой
// системы. Предназначена для разработки и тестов; в продакшене используется
// S3-совместимое хранилище (MinIO).
type LocalFileStorage struct {
	baseDir string
}

// NewLocalFileStorage создаёт хранилище в указанном каталоге.
func NewLocalFileStorage(baseDir string) (*LocalFileStorage, error) {
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, err
	}
	return &LocalFileStorage{baseDir: baseDir}, nil
}

func (s *LocalFileStorage) path(key string) string {
	return filepath.Join(s.baseDir, filepath.FromSlash(key))
}

func (s *LocalFileStorage) Upload(_ context.Context, key string, r io.Reader, _ string) error {
	p := s.path(key)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}

func (s *LocalFileStorage) Open(_ context.Context, key string) (io.ReadSeekCloser, error) {
	f, err := os.Open(s.path(key))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return f, nil
}

func (s *LocalFileStorage) Delete(_ context.Context, key string) error {
	err := os.Remove(s.path(key))
	if err != nil && os.IsNotExist(err) {
		return nil // уже удалён — идемпотентность
	}
	return err
}

// PresignedGetURL для локальной ФС не поддерживается — скачивание идёт через
// API-эндпоинт, который стримит файл из хранилища.
func (s *LocalFileStorage) PresignedGetURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "", ErrNotSupported
}
