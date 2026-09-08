package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newRequestWithIP создаёт запрос с заданным RemoteAddr (rate limiter
// идентифицирует клиента по IP).
func newRequestWithIP(ip string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.RemoteAddr = ip + ":12345"
	return req
}

func TestRateLimiter_BlocksAfterLimit(t *testing.T) {
	rl := NewRateLimiter(3, time.Minute)
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// 3 запроса проходят, 4-й блокируется.
	for i := 1; i <= 3; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, newRequestWithIP("10.0.0.1"))
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i, rec.Code)
		}
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, newRequestWithIP("10.0.0.1"))
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 on 4th request, got %d", rec.Code)
	}
}

func TestRateLimiter_IndependentIPs(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute)
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, newRequestWithIP("10.0.0.1"))
	if rec1.Code != http.StatusOK {
		t.Fatalf("first IP: expected 200, got %d", rec1.Code)
	}

	// Другой IP — свой лимит.
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, newRequestWithIP("10.0.0.2"))
	if rec2.Code != http.StatusOK {
		t.Fatalf("second IP: expected 200, got %d", rec2.Code)
	}

	// Первый IP исчерпан.
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, newRequestWithIP("10.0.0.1"))
	if rec3.Code != http.StatusTooManyRequests {
		t.Fatalf("first IP second request: expected 429, got %d", rec3.Code)
	}
}

func TestRateLimiter_WindowResets(t *testing.T) {
	rl := NewRateLimiter(1, 30*time.Millisecond)
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, newRequestWithIP("10.1.1.1"))
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec1.Code)
	}

	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, newRequestWithIP("10.1.1.1"))
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 inside window, got %d", rec2.Code)
	}

	// Ждём истечения окна — лимит сбрасывается.
	time.Sleep(40 * time.Millisecond)

	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, newRequestWithIP("10.1.1.1"))
	if rec3.Code != http.StatusOK {
		t.Fatalf("expected 200 after window reset, got %d", rec3.Code)
	}
}
