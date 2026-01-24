package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Event struct {
	ID                uuid.UUID  `json:"id"`
	OrganizerID       *uuid.UUID `json:"organizer_id,omitempty"`
	Title             string     `json:"title"`
	ShortDescription  *string    `json:"short_description,omitempty"`
	FullDescription   *string    `json:"full_description,omitempty"`
	CategoryID        *int       `json:"category_id,omitempty"`
	Importance        string     `json:"importance"`
	Status            string     `json:"status"`
	Capacity          *int       `json:"capacity,omitempty"`
	ParticipantsCount int        `json:"participants_count"`
	StartsAt          *time.Time `json:"starts_at,omitempty"`
	EndsAt            *time.Time `json:"ends_at,omitempty"`
	Address           *string    `json:"address,omitempty"`
	Latitude          *float64   `json:"latitude,omitempty"`
	Longitude         *float64   `json:"longitude,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type EventListQuery struct {
	From     *time.Time
	To       *time.Time
	City     string
	Category *int
	Status   string
	Q        string
	Page     int
	Limit    int
}

func (q *EventListQuery) Normalize() {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.Limit <= 0 || q.Limit > 100 {
		q.Limit = 20
	}
	if q.Status == "" {
		q.Status = "active"
	}
}

type eventRepository struct {
	db *pgxpool.Pool
}

func NewEventRepository(db *pgxpool.Pool) EventRepository {
	return &eventRepository{db: db}
}

func (r *eventRepository) GetByID(ctx context.Context, id uuid.UUID) (Event, error) {
	var e Event
	err := r.db.QueryRow(ctx, `
		select
			id, organizer_id, title, short_description, full_description, category_id,
			importance, status, capacity, participants_count,
			starts_at, ends_at, address, latitude, longitude, created_at, updated_at
		from events
		where id=$1 and deleted_at is null
	`, id).Scan(
		&e.ID, &e.OrganizerID, &e.Title, &e.ShortDescription, &e.FullDescription, &e.CategoryID,
		&e.Importance, &e.Status, &e.Capacity, &e.ParticipantsCount,
		&e.StartsAt, &e.EndsAt, &e.Address, &e.Latitude, &e.Longitude, &e.CreatedAt, &e.UpdatedAt,
	)
	return e, err
}

func (r *eventRepository) List(ctx context.Context, q EventListQuery) ([]Event, int, error) {
	q.Normalize()

	where := []string{"deleted_at is null"}
	args := make([]any, 0, 12)

	add := func(cond string, val any) {
		args = append(args, val)
		where = append(where, fmt.Sprintf(cond, len(args)))
	}

	if q.Status != "" {
		add("status = $%d", q.Status)
	}
	// дата
	if q.From != nil {
		add("starts_at >= $%d", *q.From)
	}
	if q.To != nil {
		add("starts_at <= $%d", *q.To)
	}
	// категория
	if q.Category != nil {
		add("category_id = $%d", *q.Category)
	}
	// место (в схеме нет city -> ищем по address)
	if strings.TrimSpace(q.City) != "" {
		args = append(args, "%"+strings.TrimSpace(q.City)+"%")
		n := len(args)
		where = append(where, fmt.Sprintf("(address ILIKE $%d)", n))
	}
	// поиск
	if strings.TrimSpace(q.Q) != "" {
		args = append(args, "%"+strings.TrimSpace(q.Q)+"%")
		n := len(args)
		where = append(where, fmt.Sprintf("(title ILIKE $%d OR short_description ILIKE $%d OR full_description ILIKE $%d)", n, n, n))
	}

	whereSQL := "where " + strings.Join(where, " and ")

	// total
	var total int
	if err := r.db.QueryRow(ctx, "select count(*) from events "+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (q.Page - 1) * q.Limit
	args2 := append(append([]any{}, args...), q.Limit, offset)
	limitPos := len(args2) - 1
	offsetPos := len(args2)

	sql := fmt.Sprintf(`
		select
			id, organizer_id, title, short_description, full_description, category_id,
			importance, status, capacity, participants_count,
			starts_at, ends_at, address, latitude, longitude, created_at, updated_at
		from events
		%s
		order by starts_at asc nulls last, created_at desc
		limit $%d offset $%d
	`, whereSQL, limitPos, offsetPos)

	rows, err := r.db.Query(ctx, sql, args2...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]Event, 0, q.Limit)
	for rows.Next() {
		var e Event
		if err := rows.Scan(
			&e.ID, &e.OrganizerID, &e.Title, &e.ShortDescription, &e.FullDescription, &e.CategoryID,
			&e.Importance, &e.Status, &e.Capacity, &e.ParticipantsCount,
			&e.StartsAt, &e.EndsAt, &e.Address, &e.Latitude, &e.Longitude, &e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return items, total, nil
}
