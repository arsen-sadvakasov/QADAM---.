package middleware

import (
	"net/http"
	"strings"
)

// CORS (раздел 28 спецификации: "строгий whitelist разрешённых origin").
// В development фронтенд Vite живёт на :5173 и ходит на API :8080 —
// без CORS браузер заблокирует ответы. В production фронтенд отдаётся
// nginx'ом с того же origin, и middleware просто не нужна.
//
// Использование: middleware.CORS("http://localhost:5173", ...) — список
// разрешённых origin. Origin не из списка — запрос проходит без
// CORS-заголовков, и браузер его блокирует (безопасно по умолчанию).
func CORS(allowedOrigins ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[strings.TrimRight(o, "/")] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" || !allowed[origin] {
				next.ServeHTTP(w, r)
				return
			}

			h := w.Header()
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Vary", "Origin")
			h.Set("Access-Control-Allow-Credentials", "true") // refresh-cookie
			h.Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			h.Set("Access-Control-Max-Age", "600")

			// Preflight — короткий ответ без передачи дальше.
			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
