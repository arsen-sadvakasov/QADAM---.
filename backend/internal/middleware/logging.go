// Package middleware содержит сквозные HTTP middleware: логирование,
// авторизацию, rate limiting, CORS (см. раздел 12 спецификации).
package middleware

import (
	"log"
	"net/http"
	"time"
)

// Logging логирует метод, путь, статус (косвенно) и время обработки каждого запроса.
// На этапе Project Setup это простая заглушка; в Phase 14 (Security Hardening)
// будет заменена на структурированное логирование (NFR-8).
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
