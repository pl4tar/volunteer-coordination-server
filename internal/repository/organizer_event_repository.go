package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"volunteer-coordination-server/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type organizerEventRepository struct {
	db *pgxpool.Pool
}

func NewOrganizerEventRepository(db *pgxpool.Pool) OrganizerEventRepository {
	return &organizerEventRepository{db: db}
}

// Create: создает мероприятие от организатора.
// По требованиям модерации — сразу ставим статус pending_moderation.
func (r *organizerEventRepository) Create(ctx context.Context, organizerID uuid.UUID, e domain.EventUpsert) (uuid.UUID, error) {
	const q = `
		insert into events(
			organizer_id, title, short_description, full_description,
			category_id, importance, status, capacity,
			starts_at, ends_at, address, latitude, longitude,
			created_at, updated_at
		)
		values(
			$1,$2,$3,$4,
			$5,COALESCE(NULLIF($6,''),'medium')::event_importance_t,
			'pending_moderation'::event_status_t,
			$7,$8,$9,$10,$11,$12,
			now(),now()
		)
		returning id
	`

	var id uuid.UUID
	if err := r.db.QueryRow(ctx, q,
		organizerID,
		e.Title,
		e.ShortDescription,
		e.FullDescription,
		e.CategoryID,
		e.Importance,
		e.Capacity,
		e.StartsAt,
		e.EndsAt,
		e.Address,
		e.Latitude,
		e.Longitude,
	).Scan(&id); err != nil {
		return uuid.Nil, err
	}

	// Теги (если пришли)
	if len(e.TagIDs) > 0 {
		if err := r.upsertTags(ctx, id, e.TagIDs); err != nil {
			return uuid.Nil, err
		}
	}

	return id, nil
}

// Update: редактирование мероприятия. Проверяем ownership.
// Простое правило: если event был active — любая правка возвращает его в pending_moderation.
func (r *organizerEventRepository) Update(ctx context.Context, organizerID, eventID uuid.UUID, e domain.EventUpsert) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var exists bool
	if err := tx.QueryRow(ctx,
		`select exists(select 1 from events where id=$1 and organizer_id=$2 and deleted_at is null)`,
		eventID, organizerID,
	).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}

	const q = `
		update events
		set
			title = COALESCE(NULLIF($3,''), title),
			short_description = COALESCE($4, short_description),
			full_description = COALESCE($5, full_description),
			category_id = COALESCE($6, category_id),
			importance = COALESCE(NULLIF($7,''), importance)::event_importance_t,
			capacity = COALESCE($8, capacity),
			starts_at = COALESCE($9, starts_at),
			ends_at = COALESCE($10, ends_at),
			address = COALESCE($11, address),
			latitude = COALESCE($12, latitude),
			longitude = COALESCE($13, longitude),
			status = case when status = 'active' then 'pending_moderation' else status end,
			updated_at = now()
		where id=$1 and organizer_id=$2 and deleted_at is null
	`

	ct, err := tx.Exec(ctx, q,
		eventID,
		organizerID,
		e.Title,
		e.ShortDescription,
		e.FullDescription,
		e.CategoryID,
		e.Importance,
		e.Capacity,
		e.StartsAt,
		e.EndsAt,
		e.Address,
		e.Latitude,
		e.Longitude,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}

	// tagIDs:
	// - nil  -> не трогаем теги
	// - []   -> очищаем теги
	// - [..] -> заменяем на новый набор
	//
	// В вашем EventUpsert TagIDs имеет тип []int (не *[]) — значит "nil" возможен только если поле отсутствует в JSON.
	// Gin binding оставит nil если поле вообще не передали.
	if e.TagIDs != nil {
		if err := r.replaceTagsTx(ctx, tx, eventID, e.TagIDs); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// SoftDelete: мягкое удаление (deleted_at) + перевод в cancelled.
// Плюс best-effort уведомление участников через DB функцию notify_event_participants.
func (r *organizerEventRepository) SoftDelete(ctx context.Context, organizerID, eventID uuid.UUID) error {
	ct, err := r.db.Exec(ctx, `
		update events
		set deleted_at = now(), status='cancelled', updated_at=now()
		where id=$1 and organizer_id=$2 and deleted_at is null
	`, eventID, organizerID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}

	// best-effort notify
	payload := map[string]any{"message": "Мероприятие отменено"}
	b, _ := json.Marshal(payload)
	_, _ = r.db.Exec(ctx,
		`select notify_event_participants($1,$2,$3::jsonb,true,true)`,
		eventID,
		"event_cancelled",
		string(b),
	)

	return nil
}

// ListByOrganizer: список мероприятий конкретного организатора (для кабинета).
func (r *organizerEventRepository) ListByOrganizer(ctx context.Context, organizerID uuid.UUID, q EventListQuery) ([]Event, int, error) {
	q.Normalize()
	q.Status = strings.TrimSpace(q.Status)

	where := []string{"deleted_at is null", "organizer_id = $1"}
	args := []any{organizerID}

	idx := func() int { return len(args) }
	add := func(cond string, val any) {
		args = append(args, val)
		where = append(where, fmt.Sprintf(cond, idx()))
	}

	if q.Status != "" {
		add("status = $%d", q.Status)
	}
	if q.From != nil {
		add("starts_at >= $%d", *q.From)
	}
	if q.To != nil {
		add("starts_at <= $%d", *q.To)
	}
	if q.Category != nil {
		add("category_id = $%d", *q.Category)
	}

	whereSQL := "where " + strings.Join(where, " and ")

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
	return items, total, rows.Err()
}

func (r *organizerEventRepository) upsertTags(ctx context.Context, eventID uuid.UUID, tagIDs []int) error {
	if len(tagIDs) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, tid := range tagIDs {
		batch.Queue(
			`insert into event_tags(event_id, tag_id) values ($1,$2) on conflict do nothing`,
			eventID, tid,
		)
	}
	br := r.db.SendBatch(ctx, batch)
	defer br.Close()

	for range tagIDs {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

func (r *organizerEventRepository) replaceTagsTx(ctx context.Context, tx pgx.Tx, eventID uuid.UUID, tagIDs []int) error {
	if _, err := tx.Exec(ctx, `delete from event_tags where event_id=$1`, eventID); err != nil {
		return err
	}
	for _, tid := range tagIDs {
		if _, err := tx.Exec(ctx,
			`insert into event_tags(event_id, tag_id) values($1,$2) on conflict do nothing`,
			eventID, tid,
		); err != nil {
			return err
		}
	}
	return nil
}
