package domain

import (
	"time"

	"github.com/google/uuid"
)

// OrganizerVerification — представление заявки на статус организатора
// для административной панели (list/details).
//
// Источник данных:
// - organizer_verifications (ov.*)
// - users (email, full_name)
//
// Важно:
// - Documents храним как jsonb в БД, наружу отдаём как map[string]any.
// - CompanyName может быть пустым/NULL.
// - ReviewedBy/ReviewedAt будут NULL пока заявка pending.
type OrganizerVerification struct {
	UserID uuid.UUID `json:"user_id"`

	// Данные пользователя (из users)
	Email    string `json:"email"`
	FullName string `json:"full_name"`

	// Данные заявки (из organizer_verifications)
	CompanyName *string        `json:"company_name,omitempty"`
	Documents   map[string]any `json:"documents,omitempty"`

	// pending | approved | rejected
	Status string `json:"status"`

	ReviewedBy *uuid.UUID `json:"reviewed_by,omitempty"`
	ReviewedAt *time.Time `json:"reviewed_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}

// Category — категория мероприятий.
// Соответствует таблице categories:
//   - id (serial/bigserial)
//   - name (text)
//   - slug (text, уникальный обычно)
//   - description (nullable)
//   - created_at (timestamptz)
type Category struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type Tag struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
