package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Review struct {
	ID        uuid.UUID  `json:"id"`
	EventID   uuid.UUID  `json:"event_id"`
	AuthorID  *uuid.UUID `json:"author_id,omitempty"`
	Rating    int16      `json:"rating"`
	Comment   *string    `json:"comment,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	IsVisible bool       `json:"is_visible"`
}

type reviewRepository struct {
	db *pgxpool.Pool
}

func NewReviewRepository(db *pgxpool.Pool) ReviewRepository {
	return &reviewRepository{db: db}
}

func (r *reviewRepository) Create(ctx context.Context, eventID, userID uuid.UUID, rating int16, comment string) error {
	if rating < 1 || rating > 5 {
		return fmt.Errorf("rating must be 1..5")
	}

	_, err := r.db.Exec(ctx, `
		insert into reviews(event_id, author_id, rating, comment, is_visible)
		values($1,$2,$3,$4,true)
		on conflict do nothing
	`, eventID, userID, rating, comment)
	return err
}
