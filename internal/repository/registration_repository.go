package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Registration struct {
	ID        uuid.UUID `json:"id"`
	EventID   uuid.UUID `json:"event_id"`
	UserID    uuid.UUID `json:"user_id"`
	Status    string    `json:"status"`
	AppliedAt time.Time `json:"applied_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Comment   *string   `json:"comment,omitempty"`
}

type registrationRepository struct {
	db *pgxpool.Pool
}

func NewRegistrationRepository(db *pgxpool.Pool) RegistrationRepository {
	return &registrationRepository{db: db}
}

func (r *registrationRepository) UpsertApplied(ctx context.Context, eventID, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		insert into registrations(event_id, user_id, status)
		values ($1,$2,'applied')
		on conflict (event_id, user_id) do update
		set status = case
			when registrations.status in ('cancelled','rejected') then 'applied'
			else registrations.status
		end,
		updated_at = now()
	`, eventID, userID)
	return err
}

func (r *registrationRepository) Cancel(ctx context.Context, eventID, userID uuid.UUID) error {
	ct, err := r.db.Exec(ctx, `
		update registrations
		set status='cancelled', updated_at=now()
		where event_id=$1 and user_id=$2 and status <> 'cancelled'
	`, eventID, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("nothing to cancel")
	}
	return nil
}

func (r *registrationRepository) CountConfirmed(ctx context.Context, eventID uuid.UUID) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `
		select count(*) from registrations
		where event_id=$1 and status in ('confirmed','attended')
	`, eventID).Scan(&n)
	return n, err
}

type HistoryItem struct {
	EventID   uuid.UUID  `json:"event_id"`
	Title     string     `json:"title"`
	Status    string     `json:"status"`
	StartsAt  *time.Time `json:"starts_at,omitempty"`
	EndsAt    *time.Time `json:"ends_at,omitempty"`
	Address   *string    `json:"address,omitempty"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (r *registrationRepository) History(ctx context.Context, userID uuid.UUID, limit, offset int) ([]HistoryItem, int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `select count(*) from registrations where user_id=$1`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		select e.id, e.title, r.status, e.starts_at, e.ends_at, e.address, r.updated_at
		from registrations r
		join events e on e.id = r.event_id
		where r.user_id=$1 and e.deleted_at is null
		order by r.updated_at desc
		limit $2 offset $3
	`, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]HistoryItem, 0, limit)
	for rows.Next() {
		var it HistoryItem
		if err := rows.Scan(&it.EventID, &it.Title, &it.Status, &it.StartsAt, &it.EndsAt, &it.Address, &it.UpdatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, it)
	}
	return items, total, rows.Err()
}
