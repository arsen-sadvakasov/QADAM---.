package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"sync"
	"time"
)

// RateLimiter — простой in-memory rate limiter по IP-адресу клиента
// (фиксированное окно). Используется для защиты /auth/login от брутфорса
// (раздел 15 спецификации: "Защита от брутфорса: rate limiting на /auth/login").
//
// В MVP достаточно in-memory реализации на один инстанс backend; при
// горизонтальном масштабировании (несколько инстансов) стоит перенести
// счётчики в Redis — уже заложен в архитектуре (раздел 10).
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	limit    int
	window   time.Duration
}

type visitor struct {
	count     int
	windowEnd time.Time
}

// NewRateLimiter создаёт лимитер, разрешающий не более limit запросов за window.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		visitors: make(map[string]*visitor),
		limit:    limit,
		window:   window,
	}
}

// Middleware оборачивает хендлер, отклоняя запросы сверх лимита с 429.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)

		rl.mu.Lock()
		v, exists := rl.visitors[ip]
		now := time.Now()
		if !exists || now.After(v.windowEnd) {
			v = &visitor{count: 0, windowEnd: now.Add(rl.window)}
			rl.visitors[ip] = v
		}
		v.count++
		exceeded := v.count > rl.limit
		rl.mu.Unlock()

		if exceeded {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "too many requests, please try again later"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return fwd
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
