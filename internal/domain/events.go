package domain

import (
	"errors"
	"time"
)

type Status string

const (
	StatusDraft             Status = "draft"
	StatusPendingModeration Status = "pending_moderation"
	StatusActive            Status = "active"
	StatusCancelled         Status = "cancelled"
	StatusCompleted         Status = "completed"
	StatusRejected          Status = "rejected"
)

type Event struct {
	ID                   string     `json:"id"`
	OrganizerID          string     `json:"organizer_id"`
	Title                string     `json:"title"`
	Description          string     `json:"description"`
	StartAt              time.Time  `json:"start_at"`
	EndAt                *time.Time `json:"end_at,omitempty"`
	Location             string     `json:"location"`
	CategoryID           *int64     `json:"category_id,omitempty"`
	RequiredParticipants int        `json:"required_participants"`
	Status               Status     `json:"status"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
	ModeratedAt          *time.Time `json:"moderated_at,omitempty"`
	ModeratedBy          *string    `json:"moderated_by,omitempty"`
}

type EventUpsert struct {
	Title            string  `json:"title"`
	ShortDescription *string `json:"short_description,omitempty"`
	FullDescription  *string `json:"full_description,omitempty"`

	CategoryID *int64 `json:"category_id,omitempty"`

	Importance string `json:"importance,omitempty"`

	Capacity *int `json:"capacity,omitempty"`

	StartsAt *time.Time `json:"starts_at,omitempty"`
	EndsAt   *time.Time `json:"ends_at,omitempty"`

	Address   *string  `json:"address,omitempty"`
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`

	TagIDs []int `json:"tag_ids,omitempty"`
}

type CreateEventRequest struct {
	Title                string     `json:"title"`
	Description          string     `json:"description"`
	StartAt              time.Time  `json:"start_at"`
	EndAt                *time.Time `json:"end_at,omitempty"`
	Location             string     `json:"location"`
	CategoryID           *int64     `json:"category_id,omitempty"`
	RequiredParticipants int        `json:"required_participants"`
}

type UpdateEventRequest struct {
	Title                *string     `json:"title,omitempty"`
	Description          *string     `json:"description,omitempty"`
	StartAt              *time.Time  `json:"start_at,omitempty"`
	EndAt                **time.Time `json:"end_at,omitempty"` // nil = не трогаем, &nil = очистить, &time = поставить
	Location             *string     `json:"location,omitempty"`
	CategoryID           **int64     `json:"category_id,omitempty"` // аналогично EndAt
	RequiredParticipants *int        `json:"required_participants,omitempty"`
}

type CreateEventResponse struct {
	Event Event `json:"event"`
}

type UpdateEventResponse struct {
	Event Event `json:"event"`
}

type DeleteEventResponse struct {
	OK bool `json:"ok"`
}

var (
	ErrForbidden       = errors.New("forbidden")
	ErrNotFound        = errors.New("not found")
	ErrInvalidArgument = errors.New("invalid argument")
)
