package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"volunteer-coordination-server/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type moderationRepository struct{ db *pgxpool.Pool }

func NewModerationRepository(db *pgxpool.Pool) ModerationRepository {
	return &moderationRepository{db: db}
}

func (r *moderationRepository) SetEventStatus(ctx context.Context, adminID, eventID uuid.UUID, status string, details map[string]any) error {
	dbDetails, _ := json.Marshal(details)

	ct, err := r.db.Exec(ctx, `
		update events
		set status=$2::event_status_t,
			moderated_by=$1,
			moderated_at=now(),
			updated_at=now()
		where id=$3 and deleted_at is null
	`, adminID, status, eventID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}

	_, _ = r.db.Exec(ctx, `
		insert into moderation_logs(object_type, object_id, action, details, performed_by)
		values('event',$1,'set_status',$2::jsonb,$3)
	`, eventID, string(dbDetails), adminID)

	return nil
}

// ---- Categories ----

type categoryRepository struct{ db *pgxpool.Pool }

func NewCategoryRepository(db *pgxpool.Pool) CategoryRepository {
	return &categoryRepository{db: db}
}

func (r *categoryRepository) List(ctx context.Context) ([]domain.Category, error) {
	rows, err := r.db.Query(ctx, `select id,name,slug,description,created_at from categories order by id asc`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.Category{}
	for rows.Next() {
		var c domain.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.Description, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *categoryRepository) Create(ctx context.Context, name, slug, description string) (int, error) {
	var id int
	var desc *string
	if description != "" {
		desc = &description
	}
	if err := r.db.QueryRow(ctx, `insert into categories(name,slug,description) values($1,$2,$3) returning id`,
		name, slug, desc,
	).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (r *categoryRepository) Delete(ctx context.Context, id int) error {
	ct, err := r.db.Exec(ctx, `delete from categories where id=$1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ---- Tags ----

type tagRepository struct{ db *pgxpool.Pool }

func NewTagRepository(db *pgxpool.Pool) TagRepository {
	return &tagRepository{db: db}
}

func (r *tagRepository) List(ctx context.Context) ([]domain.Tag, error) {
	rows, err := r.db.Query(ctx, `select id,name from tags order by id asc`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.Tag{}
	for rows.Next() {
		var t domain.Tag
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *tagRepository) Create(ctx context.Context, name string) (int, error) {
	var id int
	if err := r.db.QueryRow(ctx, `insert into tags(name) values($1) returning id`, name).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (r *tagRepository) Delete(ctx context.Context, id int) error {
	ct, err := r.db.Exec(ctx, `delete from tags where id=$1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ---- Settings ----

type settingsRepository struct{ db *pgxpool.Pool }

func NewSettingsRepository(db *pgxpool.Pool) SettingsRepository {
	return &settingsRepository{db: db}
}

func (r *settingsRepository) Get(ctx context.Context, key string) (map[string]any, error) {
	var raw []byte
	if err := r.db.QueryRow(ctx, `select value from system_settings where key=$1`, key).Scan(&raw); err != nil {
		return nil, err
	}
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	return out, nil
}

func (r *settingsRepository) Set(ctx context.Context, key string, value map[string]any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		insert into system_settings(key,value,updated_at)
		values($1,$2::jsonb,now())
		on conflict(key) do update set value=excluded.value, updated_at=now()
	`, key, string(raw))
	return err
}

// ---- Stats ----

type statsRepository struct{ db *pgxpool.Pool }

func NewStatsRepository(db *pgxpool.Pool) StatsRepository {
	return &statsRepository{db: db}
}

func (r *statsRepository) SystemStats(ctx context.Context) (map[string]any, error) {
	row := r.db.QueryRow(ctx, `select total_users,total_events,active_volunteers,generated_at from mv_system_stats`)
	var totalUsers, totalEvents, activeVolunteers int64
	var generatedAt string

	if err := row.Scan(&totalUsers, &totalEvents, &activeVolunteers, &generatedAt); err != nil {
		// fallback (если mv не создана / не обновлена)
		var u, e, a int64
		if err2 := r.db.QueryRow(ctx, `
			select
				(select count(*) from users) as total_users,
				(select count(*) from events where deleted_at is null) as total_events,
				(select count(distinct user_id) from registrations where status in ('confirmed','attended')) as active_volunteers
		`).Scan(&u, &e, &a); err2 != nil {
			return nil, err
		}
		return map[string]any{"total_users": u, "total_events": e, "active_volunteers": a}, nil
	}

	return map[string]any{
		"total_users":       totalUsers,
		"total_events":      totalEvents,
		"active_volunteers": activeVolunteers,
		"generated_at":      generatedAt,
	}, nil
}

func (r *statsRepository) OrganizerStats(ctx context.Context, organizerID uuid.UUID) (map[string]any, error) {
	var totalEvents, activeEvents, finishedEvents int64
	if err := r.db.QueryRow(ctx, `
		select
			count(*) as total,
			count(*) filter (where status='active') as active,
			count(*) filter (where status='finished') as finished
		from events
		where organizer_id=$1 and deleted_at is null
	`, organizerID).Scan(&totalEvents, &activeEvents, &finishedEvents); err != nil {
		return nil, err
	}

	var confirmed int64
	if err := r.db.QueryRow(ctx, `
		select count(*)
		from registrations r
		join events e on e.id=r.event_id
		where e.organizer_id=$1 and r.status in ('confirmed','attended') and e.deleted_at is null
	`, organizerID).Scan(&confirmed); err != nil {
		return nil, err
	}

	return map[string]any{
		"organizer_id":             organizerID,
		"total_events":             totalEvents,
		"active_events":            activeEvents,
		"finished_events":          finishedEvents,
		"confirmed_participations": confirmed,
	}, nil
}

// ---- Organizer Verification ----

type organizerVerificationRepository struct{ db *pgxpool.Pool }

func NewOrganizerVerificationRepository(db *pgxpool.Pool) OrganizerVerificationRepository {
	return &organizerVerificationRepository{db: db}
}

func (r *organizerVerificationRepository) Request(ctx context.Context, userID uuid.UUID, companyName string, documents map[string]any) error {
	docRaw, _ := json.Marshal(documents)
	var cn *string
	if companyName != "" {
		cn = &companyName
	}

	_, err := r.db.Exec(ctx, `
		insert into organizer_verifications(user_id,company_name,documents,status,created_at)
		values($1,$2,$3::jsonb,'pending',now())
		on conflict(user_id) do update
		set company_name=excluded.company_name, documents=excluded.documents, status='pending',
		    created_at=now(), reviewed_by=null, reviewed_at=null
	`, userID, cn, string(docRaw))
	if err != nil {
		return err
	}

	_, _ = r.db.Exec(ctx, `update users set organizer_status='pending' where id=$1`, userID)
	return nil
}

func (r *organizerVerificationRepository) List(ctx context.Context, status string, limit, offset int) ([]domain.OrganizerVerification, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	where := "where 1=1"
	args := []any{}
	if status != "" {
		where += fmt.Sprintf(" and ov.status=$%d", len(args)+1)
		args = append(args, status)
	}

	var total int
	if err := r.db.QueryRow(ctx, `select count(*) from organizer_verifications ov `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, `
		select ov.user_id, u.email, coalesce(u.full_name,''), ov.company_name, ov.documents,
		       ov.status, ov.reviewed_by, ov.reviewed_at, ov.created_at
		from organizer_verifications ov
		join users u on u.id=ov.user_id
		`+where+`
		order by ov.created_at desc
		limit $`+fmt.Sprint(len(args)-1)+` offset $`+fmt.Sprint(len(args))+`
	`, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]domain.OrganizerVerification, 0, limit)
	for rows.Next() {
		var it domain.OrganizerVerification
		var docRaw []byte
		if err := rows.Scan(&it.UserID, &it.Email, &it.FullName, &it.CompanyName, &docRaw,
			&it.Status, &it.ReviewedBy, &it.ReviewedAt, &it.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		if len(docRaw) > 0 {
			_ = json.Unmarshal(docRaw, &it.Documents)
		}
		out = append(out, it)
	}

	return out, total, rows.Err()
}

func (r *organizerVerificationRepository) Decide(ctx context.Context, adminID, userID uuid.UUID, approve bool, note string) error {
	newStatus := "rejected"
	if approve {
		newStatus = "approved"
	}

	details := map[string]any{"note": note, "decision": newStatus}
	raw, _ := json.Marshal(details)

	ct, err := r.db.Exec(ctx, `
		update organizer_verifications
		set status=$1, reviewed_by=$2, reviewed_at=now()
		where user_id=$3
	`, newStatus, adminID, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}

	_, err = r.db.Exec(ctx, `update users set organizer_status=$1 where id=$2`, newStatus, userID)
	if err != nil {
		return err
	}

	if approve {
		_, _ = r.db.Exec(ctx, `
			insert into user_roles(user_id,role,granted_at)
			values($1,'organizer',now())
			on conflict do nothing
		`, userID)
	}

	_, _ = r.db.Exec(ctx, `
		insert into moderation_logs(object_type,object_id,action,details,performed_by)
		values('user',$1,'organizer_verification',$2::jsonb,$3)
	`, userID, string(raw), adminID)

	return nil
}
