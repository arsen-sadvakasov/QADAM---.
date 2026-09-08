package middleware

import "net/http"

// SecurityHeaders добавляет защитные HTTP-заголовки ко всем ответам
// (Phase 14 — Security Hardening, раздел 28 спецификации).
//
//   - X-Content-Type-Options: nosniff       — запрет MIME-sniffing
//   - X-Frame-Options: DENY                 — защита от clickjacking (API не фреймится)
//   - Referrer-Policy                       — не утечь URL с токенами в реферерах
//   - Strict-Transport-Security             — только в production (HTTPS);
//     в dev HTTP без TLS, заголовок добавляется по флагу
//   - Content-Security-Policy               — для API ответов минимальная политика
func SecurityHeaders(enableHSTS bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "no-referrer")
			// API не отдаёт HTML; политика блокирует встраивание и лишние источники.
			h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
			if enableHSTS {
				h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	}
}
