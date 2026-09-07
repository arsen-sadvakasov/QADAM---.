package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/qadam/backend/internal/middleware"
	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
	"github.com/qadam/backend/internal/services"
)

// fakeNotificationRepository — in-memory реализация
// repositories.NotificationRepository для unit-тестов хендлера.
type fakeNotificationRepository struct {
	notifications map[string]*models.Notification
	recipients    map[string]*models.NotificationRecipient
	nextID        int
}

func (f *fakeNotificationRepository) Create(_ context.Context, n *models.Notification, recipientUserIDs []string) (string, error) {
	f.nextID++
	n.ID = string(rune('a' + f.nextID - 1))
	clone := *n
	f.notifications[n.ID] = &clone
	for _, uid := range recipientUserIDs {
		key := n.ID + "|" + uid
		if _, exists := f.recipients[key]; exists {
			continue
		}
		f.recipients[key] = &models.NotificationRecipient{
			ID:             "rec-" + key,
			NotificationID: n.ID,
			UserID:         uid,
		}
	}
	return n.ID, nil
}

func (f *fakeNotificationRepository) ListForUser(_ context.Context, userID string, isRead *bool, limit int) ([]*models.UserNotification, error) {
	var result []*models.UserNotification
	for _, rec := range f.recipients {
		if rec.UserID != userID {
			continue
		}
		if isRead != nil && rec.IsRead != *isRead {
			continue
		}
		n := f.notifications[rec.NotificationID]
		result = append(result, &models.UserNotification{
			Notification: *n,
			RecipientID:  rec.ID,
			IsRead:       rec.IsRead,
			ReadAt:       rec.ReadAt,
		})
	}
	return result, nil
}

func (f *fakeNotificationRepository) FindRecipient(_ context.Context, notificationID, userID string) (*models.NotificationRecipient, error) {
	key := notificationID + "|" + userID
	if rec, ok := f.recipients[key]; ok {
		return rec, nil
	}
	return nil, repositories.ErrNotFound
}

func (f *fakeNotificationRepository) MarkRead(_ context.Context, notificationID, userID string, readAt time.Time) error {
	rec, err := f.FindRecipient(context.Background(), notificationID, userID)
	if err != nil {
		return err
	}
	rec.IsRead = true
	rec.ReadAt = &readAt
	return nil
}

func (f *fakeNotificationRepository) CountUnread(_ context.Context, userID string) (int, error) {
	count := 0
	for _, rec := range f.recipients {
		if rec.UserID == userID && !rec.IsRead {
			count++
		}
	}
	return count, nil
}

func newNotificationsTestHandler() (*NotificationsHandler, *fakeNotificationRepository) {
	repo := &fakeNotificationRepository{
		notifications: make(map[string]*models.Notification),
		recipients:    make(map[string]*models.NotificationRecipient),
	}
	return NewNotificationsHandler(services.NewNotificationService(repo)), repo
}

func TestNotificationsList_Success(t *testing.T) {
	h, repo := newNotificationsTestHandler()

	// создаём уведомление для user-1
	repo.Create(context.Background(), &models.Notification{
		Type: models.NotificationSystem, Title: "Hello", Body: "World",
		Priority: models.NotificationPriorityNormal,
	}, []string{"user-1"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
	req = req.WithContext(middleware.ContextWithUser(req.Context(), "user-1", models.RoleStudent))
	rec := httptest.NewRecorder()

	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Notifications []userNotificationDTO `json:"notifications"`
		UnreadCount   int                   `json:"unread_count"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(resp.Notifications) != 1 || resp.Notifications[0].Title != "Hello" {
		t.Errorf("unexpected notifications: %+v", resp.Notifications)
	}
	if resp.UnreadCount != 1 {
		t.Errorf("expected unread_count=1, got %d", resp.UnreadCount)
	}
}

func TestNotificationsList_InvalidIsRead(t *testing.T) {
	h, _ := newNotificationsTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications?is_read=yes", nil)
	req = req.WithContext(middleware.ContextWithUser(req.Context(), "user-1", models.RoleStudent))
	rec := httptest.NewRecorder()

	h.List(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid is_read, got %d", rec.Code)
	}
}

func TestNotificationsMarkRead_SuccessAndForbidden(t *testing.T) {
	h, repo := newNotificationsTestHandler()

	id, _ := repo.Create(context.Background(), &models.Notification{
		Type: models.NotificationSystem, Title: "T", Body: "B",
		Priority: models.NotificationPriorityNormal,
	}, []string{"user-1"})

	// чужой пользователь — 403
	foreign := httptest.NewRequest(http.MethodPatch, "/api/v1/notifications/"+id+"/read", nil)
	foreign.SetPathValue("id", id)
	foreign = foreign.WithContext(middleware.ContextWithUser(foreign.Context(), "user-2", models.RoleStudent))
	foreignRec := httptest.NewRecorder()
	h.MarkRead(foreignRec, foreign)
	if foreignRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for foreign user, got %d", foreignRec.Code)
	}

	// владелец — 204
	own := httptest.NewRequest(http.MethodPatch, "/api/v1/notifications/"+id+"/read", nil)
	own.SetPathValue("id", id)
	own = own.WithContext(middleware.ContextWithUser(own.Context(), "user-1", models.RoleStudent))
	ownRec := httptest.NewRecorder()
	h.MarkRead(ownRec, own)
	if ownRec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", ownRec.Code)
	}

	// несуществующее уведомление — 403 (не раскрываем существование чужих
	// уведомлений: и «чужое», и «несуществующее» выглядят одинаково)
	missing := httptest.NewRequest(http.MethodPatch, "/api/v1/notifications/zzz/read", nil)
	missing.SetPathValue("id", "zzz")
	missing = missing.WithContext(middleware.ContextWithUser(missing.Context(), "user-1", models.RoleStudent))
	missingRec := httptest.NewRecorder()
	h.MarkRead(missingRec, missing)
	if missingRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for missing notification, got %d", missingRec.Code)
	}
}

func TestNotificationsCreate_Success(t *testing.T) {
	h, _ := newNotificationsTestHandler()

	body, _ := json.Marshal(map[string]any{
		"type":               "system",
		"title":              "Объявление",
		"body":               "Собрание в пятницу",
		"recipient_user_ids": []string{"user-1", "user-2"},
	})

	// Роль (admin-only) проверяется middleware в main.go; хендлер создаёт
	// уведомление от имени авторизованного пользователя.
	adminReq := httptest.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewReader(body))
	adminReq = adminReq.WithContext(middleware.ContextWithUser(adminReq.Context(), "admin-1", models.RoleAdmin))
	adminRec := httptest.NewRecorder()
	h.Create(adminRec, adminReq)
	if adminRec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", adminRec.Code, adminRec.Body.String())
	}
}

func TestNotificationsCreate_Validation(t *testing.T) {
	h, _ := newNotificationsTestHandler()

	// без получателей
	body, _ := json.Marshal(map[string]any{
		"type": "system", "title": "T", "body": "B",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewReader(body))
	req = req.WithContext(middleware.ContextWithUser(req.Context(), "admin-1", models.RoleAdmin))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 without recipients, got %d", rec.Code)
	}
}
