package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Notification struct {
	ID        uuid.UUID      `json:"id"`
	UserID    uuid.UUID      `json:"user_id"`
	EventID   *uuid.UUID     `json:"event_id,omitempty"`
	Type      string         `json:"type"`
	Payload   map[string]any `json:"payload,omitempty"`
	IsRead    bool           `json:"is_read"`
	CreatedAt time.Time      `json:"created_at"`
}

type notificationRepository struct {
	db *pgxpool.Pool
}

func NewNotificationRepository(db *pgxpool.Pool) NotificationRepository {
	return &notificationRepository{db: db}
}

func (r *notificationRepository) List(ctx context.Context, userID uuid.UUID, onlyUnread bool, limit int) ([]Notification, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	var (
		rows pgx.Rows
		err  error
	)

	if onlyUnread {
		rows, err = r.db.Query(ctx, `
			select id, user_id, event_id, type, payload, is_read, created_at
			from notifications
			where user_id=$1 and is_read=false
			order by created_at desc
			limit $2
		`, userID, limit)
	} else {
		rows, err = r.db.Query(ctx, `
			select id, user_id, event_id, type, payload, is_read, created_at
			from notifications
			where user_id=$1
			order by created_at desc
			limit $2
		`, userID, limit)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Notification, 0, limit)
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.EventID, &n.Type, &n.Payload, &n.IsRead, &n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}

	return out, rows.Err()
}

func (r *notificationRepository) Create(ctx context.Context, n Notification) error {
	_, err := r.db.Exec(ctx, `
		insert into notifications(user_id, event_id, type, payload, is_read)
		values($1, $2, $3, coalesce($4,'{}'::jsonb), false)
	`, n.UserID, n.EventID, n.Type, n.Payload)
	return err
}

func (r *notificationRepository) MarkRead(
	ctx context.Context,
	userID uuid.UUID,
	notificationID uuid.UUID,
) error {
	_, err := r.db.Exec(ctx, `
		update notifications
		set is_read = true
		where id = $1
		  and user_id = $2
	`, notificationID, userID)

	return err
}
