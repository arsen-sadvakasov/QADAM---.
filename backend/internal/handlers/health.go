// Package handlers содержит тонкий HTTP-слой QADAM API:
// парсинг запроса -> вызов сервиса -> формирование ответа.
package handlers

import (
	"encoding/json"
	"net/http"
	"time"
)

// HealthResponse описывает ответ health-check эндпоинта.
type HealthResponse struct {
	Status string `json:"status"`
	Time   string `json:"time"`
}

// Health отвечает 200 OK с текущим временем сервера.
// Используется для проверки доступности сервиса (см. раздел 31 DevOps спецификации).
func Health(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Status: "ok",
		Time:   time.Now().UTC().Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
