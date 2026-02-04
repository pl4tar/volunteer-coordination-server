package service

import (
	"context"
	"fmt"
	"time"

	"volunteer-coordination-server/internal/repository"

	"github.com/google/uuid"
)

type VolunteerService struct {
	events repository.EventRepository
	regs   repository.RegistrationRepository
	notif  repository.NotificationRepository
	review repository.ReviewRepository
}

func NewVolunteerService(
	events repository.EventRepository,
	regs repository.RegistrationRepository,
	notif repository.NotificationRepository,
	review repository.ReviewRepository,
) *VolunteerService {
	return &VolunteerService{events: events, regs: regs, notif: notif, review: review}
}

func (s *VolunteerService) ListEvents(ctx context.Context, q repository.EventListQuery) ([]repository.Event, int, error) {
	return s.events.List(ctx, q)
}

func (s *VolunteerService) GetEvent(ctx context.Context, id uuid.UUID) (repository.Event, error) {
	return s.events.GetByID(ctx, id)
}

func (s *VolunteerService) Signup(ctx context.Context, eventID, userID uuid.UUID) error {
	e, err := s.events.GetByID(ctx, eventID)
	if err != nil {
		return err
	}
	if e.Status != "active" {
		return fmt.Errorf("event is not active")
	}

	confirmed, err := s.regs.CountConfirmed(ctx, eventID)
	if err != nil {
		return err
	}
	if e.Capacity != nil && *e.Capacity > 0 && confirmed >= *e.Capacity {
		return fmt.Errorf("event is full")
	}

	if err := s.regs.UpsertApplied(ctx, eventID, userID); err != nil {
		return err
	}

	_ = s.notif.Create(ctx, repository.Notification{
		UserID:  userID,
		EventID: &eventID,
		Type:    "registration_applied",
		Payload: map[string]any{"message": "Заявка на участие отправлена"},
	})

	return nil
}

func (s *VolunteerService) Cancel(ctx context.Context, eventID, userID uuid.UUID) error {
	if err := s.regs.Cancel(ctx, eventID, userID); err != nil {
		return err
	}
	_ = s.notif.Create(ctx, repository.Notification{
		UserID:  userID,
		EventID: &eventID,
		Type:    "registration_cancelled",
		Payload: map[string]any{"message": "Заявка отменена"},
	})
	return nil
}

func (s *VolunteerService) Notifications(ctx context.Context, userID uuid.UUID, onlyUnread bool, limit int) ([]repository.Notification, error) {
	return s.notif.List(ctx, userID, onlyUnread, limit)
}

func (s *VolunteerService) MarkNotificationRead(ctx context.Context, userID, notificationID uuid.UUID) error {
	return s.notif.MarkRead(ctx, userID, notificationID)
}

func (s *VolunteerService) History(ctx context.Context, userID uuid.UUID, page, limit int) ([]repository.HistoryItem, int, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit
	return s.regs.History(ctx, userID, limit, offset)
}

// Простая статистика: считаем на базе registrations + events
func (s *VolunteerService) Stats(ctx context.Context, userID uuid.UUID) (map[string]any, error) {
	// Чтобы не плодить репо-функции, можно сделать позже отдельным SQL в repo.
	// Пока — на уровне сервиса минимально вернём заглушку.
	// Лучше: вынести в repository/UserStats.
	return map[string]any{
		"user_id": userID,
		"note":    "",
		"now":     time.Now(),
	}, nil
}

// Рейтинг: score = count(confirmed/attended) + sum(importance weight)
// importance: low=1, medium=2, high=3, critical=5
func (s *VolunteerService) Rating(ctx context.Context, userID uuid.UUID) (map[string]any, error) {
	// Аналогично: лучше отдельным SQL в repo. Пока отдаём “hook”.
	return map[string]any{
		"user_id": userID,
		"note":    "4.2",
	}, nil
}

func (s *VolunteerService) LeaveReview(ctx context.Context, eventID, userID uuid.UUID, rating int16, comment string) error {
	e, err := s.events.GetByID(ctx, eventID)
	if err != nil {
		return err
	}
	if e.StartsAt != nil && time.Now().Before(*e.StartsAt) {
		return fmt.Errorf("too early to leave a review")
	}
	return s.review.Create(ctx, eventID, userID, rating, comment)
}
