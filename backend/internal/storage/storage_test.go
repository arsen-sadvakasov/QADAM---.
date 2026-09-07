package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestLocalFileStorage_UploadOpenDelete(t *testing.T) {
	s, err := NewLocalFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	ctx := context.Background()

	content := []byte("hello qadam materials")
	if err := s.Upload(ctx, "materials/sub-1/abc.pdf", bytes.NewReader(content), "application/pdf"); err != nil {
		t.Fatalf("upload failed: %v", err)
	}

	rc, err := s.Open(ctx, "materials/sub-1/abc.pdf")
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	got, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if !bytes.Equal(content, got) {
		t.Errorf("content mismatch: got %q, want %q", got, content)
	}

	if err := s.Delete(ctx, "materials/sub-1/abc.pdf"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	// Повторное чтение — ErrNotFound
	_, err = s.Open(ctx, "materials/sub-1/abc.pdf")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}

	// Повторное удаление — идемпотентно
	if err := s.Delete(ctx, "materials/sub-1/abc.pdf"); err != nil {
		t.Errorf("expected idempotent delete, got %v", err)
	}
}

func TestLocalFileStorage_OpenMissing(t *testing.T) {
	s, err := NewLocalFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	_, err = s.Open(context.Background(), "missing/key.pdf")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestLocalFileStorage_PresignedNotSupported(t *testing.T) {
	s, _ := NewLocalFileStorage(t.TempDir())
	_, err := s.PresignedGetURL(context.Background(), "k", 0)
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("expected ErrNotSupported, got %v", err)
	}
}

func TestNewKey(t *testing.T) {
	key1, err := NewKey("sub-1", "Лекция 1.PDF")
	if err != nil {
		t.Fatalf("NewKey failed: %v", err)
	}
	if !strings.HasPrefix(key1, "materials/sub-1/") {
		t.Errorf("expected prefix materials/sub-1/, got %q", key1)
	}
	if !strings.HasSuffix(key1, ".pdf") {
		t.Errorf("expected lowercased extension .pdf, got %q", key1)
	}

	key2, _ := NewKey("sub-1", "Лекция 1.PDF")
	if key1 == key2 {
		t.Errorf("expected unique keys, got identical %q", key1)
	}
}
