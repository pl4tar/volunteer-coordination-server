package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"volunteer-coordination-server/internal/domain"

	"volunteer-coordination-server/internal/repository"

	"github.com/google/uuid"
)

// OrganizerService — бизнес-логика организатора.
// В этом файле — CRUD мероприятий (создание / редактирование / удаление) + список.
//
// Важно:
// - Репозиторий отвечает за ownership на уровне SQL (organizer_id = $1).
// - Сервис отвечает за валидацию и бизнес-правила уровня приложения.
type OrganizerService struct {
	events repository.OrganizerEventRepository
	clock  func() time.Time
}

func NewOrganizerService(events repository.OrganizerEventRepository) *OrganizerService {
	return &OrganizerService{
		events: events,
		clock:  time.Now,
	}
}

// CreateEvent создает мероприятие организатора.
// Статус мероприятия выставляется репозиторием/БД в pending_moderation.
func (s *OrganizerService) CreateEvent(ctx context.Context, organizerID uuid.UUID, req domain.EventUpsert) (uuid.UUID, error) {
	if err := s.validateCreate(req); err != nil {
		return uuid.Nil, err
	}

	id, err := s.events.Create(ctx, organizerID, req)
	if err != nil {
		return uuid.Nil, s.mapRepoErr(err)
	}

	return id, nil
}

// UpdateEvent обновляет мероприятие (частично).
// Если мероприятие было active — репозиторий переводит в pending_moderation (правило модерации).
func (s *OrganizerService) UpdateEvent(ctx context.Context, organizerID, eventID uuid.UUID, patch domain.EventUpsert) error {
	if err := s.validateUpdate(patch); err != nil {
		return err
	}

	if err := s.events.Update(ctx, organizerID, eventID, patch); err != nil {
		return s.mapRepoErr(err)
	}

	return nil
}

// DeleteEvent выполняет мягкое удаление мероприятия:
// deleted_at=now(), status='cancelled' (делает репозиторий).
func (s *OrganizerService) DeleteEvent(ctx context.Context, organizerID, eventID uuid.UUID) error {
	if eventID == uuid.Nil {
		return ErrInvalidArgument("event_id is required")
	}

	if err := s.events.SoftDelete(ctx, organizerID, eventID); err != nil {
		return s.mapRepoErr(err)
	}

	return nil
}

// ListMyEvents возвращает список мероприятий организатора с пагинацией.
func (s *OrganizerService) ListMyEvents(ctx context.Context, organizerID uuid.UUID, q repository.EventListQuery) ([]repository.Event, int, error) {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.Limit <= 0 {
		q.Limit = 20
	}
	if q.Limit > 100 {
		q.Limit = 100
	}

	events, total, err := s.events.ListByOrganizer(ctx, organizerID, q)
	if err != nil {
		return nil, 0, s.mapRepoErr(err)
	}
	return events, total, nil
}

func (s *OrganizerService) validateCreate(req domain.EventUpsert) error {
	// Title
	if strings.TrimSpace(req.Title) == "" {
		return ErrInvalidArgument("title is required")
	}

	// Capacity (в вашем DTO это обычно *int)
	if req.Capacity == nil || *req.Capacity <= 0 {
		return ErrInvalidArgument("capacity must be > 0")
	}

	// StartsAt (обычно *time.Time)
	if req.StartsAt == nil {
		return ErrInvalidArgument("starts_at is required")
	}

	// Базовая логика: старт не должен быть сильно в прошлом
	now := s.clock()
	if req.StartsAt.Before(now.Add(-1 * time.Minute)) {
		return ErrInvalidArgument("starts_at must be in the future")
	}

	// EndsAt >= StartsAt
	if req.EndsAt != nil && req.EndsAt.Before(*req.StartsAt) {
		return ErrInvalidArgument("ends_at must be >= starts_at")
	}

	// Importance (если передали)
	if req.Importance != "" && !isAllowedImportance(req.Importance) {
		return ErrInvalidArgument("importance must be one of: low|medium|high")
	}

	// Координаты (если одна есть — должна быть и вторая)
	if (req.Latitude != nil && req.Longitude == nil) || (req.Latitude == nil && req.Longitude != nil) {
		return ErrInvalidArgument("latitude and longitude must be provided together")
	}

	// tags: отрицательные/0 нельзя
	if req.TagIDs != nil {
		for _, id := range req.TagIDs {
			if id <= 0 {
				return ErrInvalidArgument("tag_ids must contain positive integers")
			}
		}
	}

	return nil
}

func (s *OrganizerService) validateUpdate(patch domain.EventUpsert) error {
	// Update у вас "частичный", но Title может прилететь как пустая строка —
	// считаем это ошибкой (если передали).
	if patch.Title != "" && strings.TrimSpace(patch.Title) == "" {
		return ErrInvalidArgument("title cannot be empty")
	}

	if patch.Capacity != nil && *patch.Capacity <= 0 {
		return ErrInvalidArgument("capacity must be > 0")
	}

	if patch.StartsAt != nil {
		// Можно разрешить перенос в прошлое? Обычно нет.
		now := s.clock()
		if patch.StartsAt.Before(now.Add(-1 * time.Minute)) {
			return ErrInvalidArgument("starts_at must be in the future")
		}
	}

	if patch.StartsAt != nil && patch.EndsAt != nil && patch.EndsAt.Before(*patch.StartsAt) {
		return ErrInvalidArgument("ends_at must be >= starts_at")
	}

	if patch.Importance != "" && !isAllowedImportance(patch.Importance) {
		return ErrInvalidArgument("importance must be one of: low|medium|high")
	}

	if (patch.Latitude != nil && patch.Longitude == nil) || (patch.Latitude == nil && patch.Longitude != nil) {
		return ErrInvalidArgument("latitude and longitude must be provided together")
	}

	if patch.TagIDs != nil {
		for _, id := range patch.TagIDs {
			if id <= 0 {
				return ErrInvalidArgument("tag_ids must contain positive integers")
			}
		}
	}

	return nil
}

func isAllowedImportance(v string) bool {
	switch v {
	case "low", "medium", "high":
		return true
	default:
		return false
	}
}

type ServiceError struct {
	kind string
	msg  string
}

func (e *ServiceError) Error() string { return e.msg }
func (e *ServiceError) Kind() string  { return e.kind }

func ErrInvalidArgument(msg string) error { return &ServiceError{kind: "invalid_argument", msg: msg} }
func ErrNotFound(msg string) error        { return &ServiceError{kind: "not_found", msg: msg} }
func ErrForbidden(msg string) error       { return &ServiceError{kind: "forbidden", msg: msg} }
func ErrInternal(msg string) error        { return &ServiceError{kind: "internal", msg: msg} }

func (s *OrganizerService) mapRepoErr(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound("event not found")
	}

	return fmt.Errorf("%w: %v", ErrInternal("repository error"), err)
}
