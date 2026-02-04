package repository

import (
	"context"
	"errors"
	"time"
	"volunteer-coordination-server/internal/domain"

	"github.com/google/uuid"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
)

// UserRepository интерфейс для работы с пользователями
type UserRepository interface {
	Save(ctx context.Context, user domain.User) (uuid.UUID, error)
	QueryGetUserByEmail(ctx context.Context, email string) (domain.User, error)
	QueryGetUserByID(ctx context.Context, userID uuid.UUID) (domain.User, error)
	//GetUserRoleByID(ctx context.Context, userID uuid.UUID) (domain.UserRole, error)

	ExistsByEmail(ctx context.Context, email string) (bool, error)
	ListUsers(ctx context.Context, page, limit int, roleFilter string) ([]domain.User, int64, error)

	UpdateProfile(ctx context.Context, user domain.User) error
	UpdatePassword(ctx context.Context, userID uuid.UUID, newPasswordHash string) error

	AddUserRole(ctx context.Context, userID uuid.UUID, role string) error
	RemoveUserRole(ctx context.Context, userID uuid.UUID, role string) error

	BlockUser(ctx context.Context, userID uuid.UUID) error
	UnblockUser(ctx context.Context, userID uuid.UUID) error
}

// SessionRepository интерфейс для работы с сессиями
type SessionRepository interface {
	Save(ctx context.Context, session *domain.Session) error
	UpdateUserSession(ctx context.Context, refreshToken string, expiresAt time.Time, sessionID uuid.UUID) error
	GetSessionByRefreshToken(ctx context.Context, refreshToken string) (domain.Session, error)
	Delete(ctx context.Context, sessionID uuid.UUID) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error

	FindByID(ctx context.Context, id uuid.UUID) (domain.Session, error)
	DeleteExpired(ctx context.Context) error
}

// EventRepository интерфейс для мероприятий
type EventRepository interface {
	List(ctx context.Context, q EventListQuery) ([]Event, int, error)
	GetByID(ctx context.Context, id uuid.UUID) (Event, error)
}

// RegistrationRepository интерфейс для заявок
type RegistrationRepository interface {
	UpsertApplied(ctx context.Context, eventID, userID uuid.UUID) error
	Cancel(ctx context.Context, eventID, userID uuid.UUID) error
	CountConfirmed(ctx context.Context, eventID uuid.UUID) (int, error)
	History(ctx context.Context, userID uuid.UUID, limit, offset int) ([]HistoryItem, int, error)
}

// NotificationRepository интерфейс для уведомлений
type NotificationRepository interface {
	List(ctx context.Context, userID uuid.UUID, onlyUnread bool, limit int) ([]Notification, error)
	Create(ctx context.Context, n Notification) error
	MarkRead(ctx context.Context, userID, notificationID uuid.UUID) error
}

// ReviewRepository интерфейс для отзывов
type ReviewRepository interface {
	Create(ctx context.Context, eventID, userID uuid.UUID, rating int16, comment string) error
}

// OrganizerEventRepository
//
// Отвечает ТОЛЬКО за работу организатора со своими мероприятиями.
// Никакой бизнес-логики, только доступ к данным.
//
// Основные инварианты:
//   - Организатор может работать ТОЛЬКО со своими событиями
//   - Удаление является soft-delete (deleted_at + status=cancelled)
//   - Любое изменение active-события возвращает его в pending_moderation
type OrganizerEventRepository interface {

	// Create создает новое мероприятие от имени организатора.
	//
	// Поведение:
	//   - organizerID — идентификатор пользователя-организатора
	//   - статус события всегда выставляется в `pending_moderation`
	//   - created_at / updated_at выставляются на уровне БД
	//   - теги (tag_ids) добавляются в таблицу event_tags, если они переданы
	//
	// Возвращает:
	//   - uuid созданного события
	//   - ошибку БД при неуспехе
	//
	Create(
		ctx context.Context,
		organizerID uuid.UUID,
		event domain.EventUpsert,
	) (uuid.UUID, error)

	// Update обновляет мероприятие, принадлежащее организатору.
	//
	// Правила:
	//   - обновление возможно ТОЛЬКО если event принадлежит organizerID
	//   - deleted_at IS NULL
	//   - если статус события был `active`, он переводится в `pending_moderation`
	//
	// Частичное обновление:
	//   - поля, равные nil / zero-value (в зависимости от DTO), НЕ изменяются
	//   - TagIDs:
	//       nil  -> не трогаем теги
	//       []   -> очищаем все теги
	//       [..] -> заменяем набор тегов
	//
	// Возвращает:
	//   - ErrNotFound, если событие не найдено или не принадлежит организатору
	//   - ошибку БД при неуспехе
	//
	Update(
		ctx context.Context,
		organizerID uuid.UUID,
		eventID uuid.UUID,
		event domain.EventUpsert,
	) error

	// SoftDelete выполняет мягкое удаление мероприятия.
	//
	// Поведение:
	//   - выставляет deleted_at = now()
	//   - выставляет status = 'cancelled'
	//   - обновляет updated_at
	//   - операция идемпотентна (повторный вызов вернет ErrNotFound)
	//
	// Дополнительно (best-effort):
	//   - может отправлять уведомление участникам через DB-функцию
	//     notify_event_participants(...)
	//
	// Возвращает:
	//   - ErrNotFound, если событие не найдено или не принадлежит организатору
	//   - ошибку БД при неуспехе
	//
	SoftDelete(
		ctx context.Context,
		organizerID uuid.UUID,
		eventID uuid.UUID,
	) error

	// ListByOrganizer возвращает список мероприятий конкретного организатора
	// с поддержкой фильтрации и пагинации (кабинет организатора).
	//
	// Фильтры (см. EventListQuery):
	//   - статус события
	//   - диапазон дат
	//   - категория
	//
	// Пагинация:
	//   - Page (1-based)
	//   - Limit
	//
	// Возвращает:
	//   - список событий
	//   - общее количество (для фронта)
	//   - ошибку БД при неуспехе
	//
	ListByOrganizer(
		ctx context.Context,
		organizerID uuid.UUID,
		query EventListQuery,
	) (events []Event, total int, err error)
}

// ModerationRepository — модерация (в первую очередь мероприятий).
type ModerationRepository interface {
	// SetEventStatus меняет статус события (active/rejected/pending_moderation/...)
	// + пишет запись в moderation_logs.
	SetEventStatus(ctx context.Context, adminID, eventID uuid.UUID, status string, details map[string]any) error
}

// CategoryRepository — управление категориями мероприятий.
type CategoryRepository interface {
	List(ctx context.Context) ([]domain.Category, error)
	Create(ctx context.Context, name, slug, description string) (int, error)
	Delete(ctx context.Context, id int) error
}

// TagRepository — управление тегами мероприятий.
type TagRepository interface {
	List(ctx context.Context) ([]domain.Tag, error)
	Create(ctx context.Context, name string) (int, error)
	Delete(ctx context.Context, id int) error
}

// SettingsRepository — системные настройки/интеграции (jsonb).
type SettingsRepository interface {
	Get(ctx context.Context, key string) (map[string]any, error)
	Set(ctx context.Context, key string, value map[string]any) error
}

// StatsRepository — статистика системы/организатора.
type StatsRepository interface {
	SystemStats(ctx context.Context) (map[string]any, error)
	OrganizerStats(ctx context.Context, organizerID uuid.UUID) (map[string]any, error)
}

// OrganizerVerificationRepository — заявки на статус организатора (pending/approved/rejected).
type OrganizerVerificationRepository interface {
	Request(ctx context.Context, userID uuid.UUID, companyName string, documents map[string]any) error
	List(ctx context.Context, status string, limit, offset int) ([]domain.OrganizerVerification, int, error)
	Decide(ctx context.Context, adminID, userID uuid.UUID, approve bool, note string) error
}
