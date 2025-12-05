package UserRepository

import "github.com/jackc/pgx/v5/pgxpool"

type UsrRepository struct {
	db *pgxpool.Pool
}

func NewUsrRepository(db *pgxpool.Pool) *UsrRepository {
	return &UsrRepository{db: db}
}
