package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/qadam/backend/internal/middleware"
	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
	"github.com/qadam/backend/internal/services"
)

// NotificationsHandler содержит HTTP-хендлеры для /api/v1/notifications/*
// (раздел 35 API Plan): список своих — все авторизованные; отметка прочтения —
// владелец уведомления; создание системного уведомления — Admin.
type NotificationsHandler struct {
	notifications *services.NotificationService
}

// NewNotificationsHandler создаёт NotificationsHandler с внедрённым сервисом.
func NewNotificationsHandler(notifications *services.NotificationService) *NotificationsHandler {
	return &NotificationsHandler{notifications: notifications}
}

type userNotificationDTO struct {
	ID                string `json:"id"`
	Type              string `json:"type"`
	Title             string `json:"title"`
	Body              string `json:"body"`
	Priority          string `json:"priority"`
	RelatedEntityType *string `json:"related_entity_type"`
	RelatedEntityID   *string `json:"related_entity_id"`
	CreatedAt         string `json:"created_at"`
	IsRead            bool   `json:"is_read"`
	ReadAt            *string `json:"read_at"`
}

func userNotificationDTOFromModel(un *models.UserNotification) userNotificationDTO {
	dto := userNotificationDTO{
		ID:                un.ID,
		Type:              string(un.Type),
		Title:             un.Title,
		Body:              un.Body,
		Priority:          string(un.Priority),
		RelatedEntityType: un.RelatedEntityType,
		RelatedEntityID:   un.RelatedEntityID,
		CreatedAt:         un.CreatedAt.Format(time.RFC3339),
		IsRead:            un.IsRead,
	}
	if un.ReadAt != nil {
		formatted := un.ReadAt.Format(time.RFC3339)
		dto.ReadAt = &formatted
	}
	return dto
}

// List обрабатывает GET /api/v1/notifications?is_read=&limit= — список
// уведомлений текущего пользователя.
func (h *NotificationsHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var isRead *bool
	if param := r.URL.Query().Get("is_read"); param != "" {
		parsed, err := strconv.ParseBool(param)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid is_read value, expected true or false")
			return
		}
		isRead = &parsed
	}

	var limit int
	if param := r.URL.Query().Get("limit"); param != "" {
		parsed, err := strconv.Atoi(param)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, "invalid limit value, expected positive integer")
			return
		}
		limit = parsed
	}

	list, err := h.notifications.ListForUser(r.Context(), userID, isRead, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	dtos := make([]userNotificationDTO, 0, len(list))
	for _, un := range list {
		dtos = append(dtos, userNotificationDTOFromModel(un))
	}

	unread, err := h.notifications.CountUnread(r.Context(), userID)
	if err != nil {
		unread = 0 // счётчик не критичен — не срываем ответ
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"notifications":  dtos,
		"unread_count": unread,
	})
}

// MarkRead обрабатывает PATCH /api/v1/notifications/{id}/read — отметить
// уведомление прочитанным (только своё).
func (h *NotificationsHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	notificationID := r.PathValue("id")

	if err := h.notifications.MarkRead(r.Context(), userID, notificationID); err != nil {
		switch {
		case errors.Is(err, services.ErrNotificationForbidden):
			writeError(w, http.StatusForbidden, err.Error())
		case errors.Is(err, repositories.ErrNotFound):
			writeError(w, http.StatusNotFound, "notification not found")
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type notificationCreateRequest struct {
	Type              string   `json:"type"`
	Title             string   `json:"title"`
	Body              string   `json:"body"`
	Priority          string   `json:"priority"`
	RelatedEntityType *string  `json:"related_entity_type"`
	RelatedEntityID   *string  `json:"related_entity_id"`
	RecipientUserIDs  []string `json:"recipient_user_ids"`
}

// Create обрабатывает POST /api/v1/notifications — создание системного
// уведомления. Admin only (раздел 35).
func (h *NotificationsHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req notificationCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.RecipientUserIDs) == 0 {
		writeError(w, http.StatusBadRequest, "recipient_user_ids is required")
		return
	}

	id, err := h.notifications.Create(r.Context(), &userID, services.NotificationInput{
		Type:              models.NotificationType(req.Type),
		Title:             req.Title,
		Body:              req.Body,
		Priority:          models.NotificationPriority(req.Priority),
		RelatedEntityType: req.RelatedEntityType,
		RelatedEntityID:   req.RelatedEntityID,
	}, req.RecipientUserIDs)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrNotificationValidation):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}
