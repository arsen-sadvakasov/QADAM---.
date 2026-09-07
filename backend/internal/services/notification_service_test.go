package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
	"github.com/qadam/backend/internal/storage"
)

// fakeNotificationRepository — in-memory реализация
// repositories.NotificationRepository для unit-тестов.
type fakeNotificationRepository struct {
	notifications map[string]*models.Notification
	recipients    map[string]*models.NotificationRecipient // ключ: notificationID|userID
	nextID        int
	created       []string // ID созданных уведомлений в порядке создания
}

func newFakeNotificationRepository() *fakeNotificationRepository {
	return &fakeNotificationRepository{
		notifications: make(map[string]*models.Notification),
		recipients:    make(map[string]*models.NotificationRecipient),
	}
}

func (f *fakeNotificationRepository) Create(_ context.Context, n *models.Notification, recipientUserIDs []string) (string, error) {
	f.nextID++
	n.ID = string(rune('a' + f.nextID - 1))
	clone := *n
	f.notifications[n.ID] = &clone
	f.created = append(f.created, n.ID)
	for _, uid := range recipientUserIDs {
		key := n.ID + "|" + uid
		if _, exists := f.recipients[key]; exists {
			continue // ON CONFLICT DO NOTHING
		}
		f.recipients[key] = &models.NotificationRecipient{
			ID:             "rec-" + key,
			NotificationID: n.ID,
			UserID:         uid,
			IsRead:         false,
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

func TestNotificationService_Create_Success(t *testing.T) {
	repo := newFakeNotificationRepository()
	svc := NewNotificationService(repo)

	id, err := svc.Create(context.Background(), nil, NotificationInput{
		Type:  models.NotificationSystem,
		Title: "Объявление",
		Body:  "Собрание в пятницу",
	}, []string{"user-1", "user-2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty notification ID")
	}
	if len(repo.created) != 1 {
		t.Errorf("expected 1 notification created, got %d", len(repo.created))
	}
}

func TestNotificationService_Create_Validation(t *testing.T) {
	repo := newFakeNotificationRepository()
	svc := NewNotificationService(repo)
	ctx := context.Background()
	recipients := []string{"user-1"}

	// неизвестный тип
	_, err := svc.Create(ctx, nil, NotificationInput{Type: "bogus", Title: "T", Body: "B"}, recipients)
	if !errors.Is(err, ErrNotificationValidation) {
		t.Errorf("expected validation error for unknown type, got %v", err)
	}

	// без title
	_, err = svc.Create(ctx, nil, NotificationInput{Type: models.NotificationSystem, Body: "B"}, recipients)
	if !errors.Is(err, ErrNotificationValidation) {
		t.Errorf("expected validation error without title, got %v", err)
	}

	// неизвестный приоритет
	_, err = svc.Create(ctx, nil, NotificationInput{Type: models.NotificationSystem, Title: "T", Body: "B", Priority: "urgent"}, recipients)
	if !errors.Is(err, ErrNotificationValidation) {
		t.Errorf("expected validation error for unknown priority, got %v", err)
	}

	// без получателей
	_, err = svc.Create(ctx, nil, NotificationInput{Type: models.NotificationSystem, Title: "T", Body: "B"}, nil)
	if !errors.Is(err, ErrNotificationValidation) {
		t.Errorf("expected validation error without recipients, got %v", err)
	}
}

func TestNotificationService_MarkRead(t *testing.T) {
	repo := newFakeNotificationRepository()
	svc := NewNotificationService(repo)
	ctx := context.Background()

	id, _ := svc.Create(ctx, nil, NotificationInput{Type: models.NotificationSystem, Title: "T", Body: "B"}, []string{"user-1"})

	// чужой пользователь не может отметить
	if err := svc.MarkRead(ctx, "user-2", id); !errors.Is(err, ErrNotificationForbidden) {
		t.Fatalf("expected ErrNotificationForbidden for foreign user, got %v", err)
	}

	// владелец отмечает
	if err := svc.MarkRead(ctx, "user-1", id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	unread, _ := svc.CountUnread(ctx, "user-1")
	if unread != 0 {
		t.Errorf("expected 0 unread after mark read, got %d", unread)
	}

	// повторная отметка — идемпотентна
	if err := svc.MarkRead(ctx, "user-1", id); err != nil {
		t.Errorf("expected idempotent mark read, got %v", err)
	}
}

func TestNotificationService_ListForUser_FilterByRead(t *testing.T) {
	repo := newFakeNotificationRepository()
	svc := NewNotificationService(repo)
	ctx := context.Background()

	id1, _ := svc.Create(ctx, nil, NotificationInput{Type: models.NotificationSystem, Title: "T1", Body: "B"}, []string{"user-1"})
	svc.Create(ctx, nil, NotificationInput{Type: models.NotificationSystem, Title: "T2", Body: "B"}, []string{"user-1"})
	_ = svc.MarkRead(ctx, "user-1", id1)

	// is_read=false — только непрочитанные (T1 уже прочитан)
	unread := false
	list, err := svc.ListForUser(ctx, "user-1", &unread, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected only 1 unread notification, got %d", len(list))
	}
	if list[0].Title != "T2" {
		t.Errorf("expected unread T2, got %q", list[0].Title)
	}
}

func TestNotificationService_NotifyScheduleChange_CancelType(t *testing.T) {
	repo := newFakeNotificationRepository()
	svc := NewNotificationService(repo)

	err := svc.NotifyScheduleChange(context.Background(), models.ScheduleChangeCancel,
		"chg-1", "PO-23", "Базы данных", "2024-03-11", []string{"user-1", "user-2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	n := repo.notifications[repo.created[0]]
	if n.Type != models.NotificationCancellation {
		t.Errorf("expected cancellation type, got %q", n.Type)
	}
	if n.Priority != models.NotificationPriorityHigh {
		t.Errorf("expected high priority, got %q", n.Priority)
	}
	if !strings.Contains(n.Body, "PO-23") || !strings.Contains(n.Body, "2024-03-11") {
		t.Errorf("expected body to contain group and date, got %q", n.Body)
	}
}

func TestNotificationService_NotifyNewMaterial(t *testing.T) {
	repo := newFakeNotificationRepository()
	svc := NewNotificationService(repo)

	err := svc.NotifyNewMaterial(context.Background(), "mat-1", "Базы данных", "Лекция 1", []string{"user-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	n := repo.notifications[repo.created[0]]
	if n.Type != models.NotificationNewMaterial {
		t.Errorf("expected new_material type, got %q", n.Type)
	}
}

// Проверка интеграции: создание замены должно отправить уведомление группе.
type spyNotifier struct {
	calls     int
	lastType  models.ScheduleChangeType
	lastUsers []string
}

func (s *spyNotifier) NotifyScheduleChange(_ context.Context, changeType models.ScheduleChangeType, _, _, _, _ string, studentUserIDs []string) error {
	s.calls++
	s.lastType = changeType
	s.lastUsers = studentUserIDs
	return nil
}

func TestScheduleChangeService_Create_SendsNotification(t *testing.T) {
	templates := newFakeScheduleTemplateRepository()
	templates.byID["template-1"] = &models.ScheduleTemplate{ID: "template-1", Status: models.ScheduleTemplateStatusActive}
	templates.studentUserIDs = []string{"user-1", "user-2"} // студенты группы
	spy := &spyNotifier{}
	svc := NewScheduleChangeService(newFakeScheduleChangeRepository(), templates, spy)

	_, err := svc.Create(context.Background(), "admin-1", baseScheduleChangeInput(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spy.calls != 1 {
		t.Fatalf("expected 1 notification call, got %d", spy.calls)
	}
	if spy.lastType != models.ScheduleChangeCancel {
		t.Errorf("expected cancel type, got %q", spy.lastType)
	}
}

func TestScheduleChangeService_Create_NotifierErrorDoesNotFailCreate(t *testing.T) {
	templates := newFakeScheduleTemplateRepository()
	templates.byID["template-1"] = &models.ScheduleTemplate{ID: "template-1", Status: models.ScheduleTemplateStatusActive}
	failing := &failingNotifier{}
	svc := NewScheduleChangeService(newFakeScheduleChangeRepository(), templates, failing)

	created, err := svc.Create(context.Background(), "admin-1", baseScheduleChangeInput(t))
	if err != nil {
		t.Fatalf("expected change creation to succeed despite notifier error, got %v", err)
	}
	if created.ID == "" {
		t.Error("expected non-empty change ID")
	}
}

type failingNotifier struct{}

func (f *failingNotifier) NotifyScheduleChange(context.Context, models.ScheduleChangeType, string, string, string, string, []string) error {
	return errors.New("notification service down")
}

// Интеграция материалов: создание материала должно отправить уведомление.
type materialSpyNotifier struct {
	calls     int
	lastTitle string
	lastUsers []string
}

func (s *materialSpyNotifier) NotifyNewMaterial(_ context.Context, _, subjectName, materialTitle string, recipientUserIDs []string) error {
	s.calls++
	s.lastTitle = materialTitle
	s.lastUsers = recipientUserIDs
	return nil
}

func TestMaterialService_Create_SendsNotification(t *testing.T) {
	repo := newFakeMaterialRepository()
	repo.subscriberUserIDs = []string{"user-1", "user-2"} // подписчики предмета
	spy := &materialSpyNotifier{}
	fs, err := storage.NewLocalFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("failed to init storage: %v", err)
	}
	svc := NewMaterialService(repo, fs, spy)

	_, err = svc.Create(context.Background(), "teacher-1", MaterialInput{
		SubjectID: "subject-1",
		Title:     "Лекция 2",
		Category:  models.MaterialCategoryLecture,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spy.calls != 1 {
		t.Fatalf("expected 1 notification call, got %d", spy.calls)
	}
	if spy.lastTitle != "Лекция 2" {
		t.Errorf("expected material title in notification, got %q", spy.lastTitle)
	}
}
