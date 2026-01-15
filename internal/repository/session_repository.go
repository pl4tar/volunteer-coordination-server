package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"volunteer-coordination-server/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type sessionRepository struct {
	db *pgxpool.Pool
}

func NewSessionRepository(db *pgxpool.Pool) SessionRepository {
	return &sessionRepository{db: db}
}

// Save создает новую сессию пользователя
func (r *sessionRepository) Save(ctx context.Context, session domain.Session) error {
	const op = "postgres.sessionRepository.Save"

	query := `
		INSERT INTO user_sessions (
			id, user_id, refresh_token, user_agent, 
			ip_address, created_at, expires_at
		) VALUES ($1, $2, $3, $4, $5::inet, NOW(), NOW() + INTERVAL '30 days')
		RETURNING created_at, expires_at
	`

	err := r.db.QueryRow(ctx, query,
		session.ID,
		session.UserID,
		session.RefreshToken,
		session.UserAgent,
		session.IPAddress,
	).Scan(&session.CreatedAt, &session.ExpiresAt)

	if err != nil {
		slog.Error("failed to create session",
			slog.String("op", op),
			slog.String("user_id", session.UserID.String()),
			slog.Any("error", err))
		return fmt.Errorf("%s: %w", op, err)
	}

	slog.Info("session created",
		slog.String("op", op),
		slog.String("user_id", session.UserID.String()),
		slog.String("session_id", session.ID.String()))

	return nil
}

// UpdateUserSession обновляет сессию с новым JWT refresh токеном
func (r *sessionRepository) UpdateUserSession(ctx context.Context, refreshToken string, sessionID uuid.UUID) error {
	const op = "postgres.sessionRepository.UpdateUserSession"

	query := `
		UPDATE user_sessions 
		SET refresh_token = $1, updated_at = NOW() 
		WHERE id = $2
	`

	result, err := r.db.Exec(ctx, query, refreshToken, sessionID)
	if err != nil {
		slog.Error("failed to update user session",
			slog.String("op", op),
			slog.String("session_id", sessionID.String()),
			slog.Any("error", err))
		return fmt.Errorf("%s: %w", op, err)
	}

	if result.RowsAffected() == 0 {
		slog.Warn("session not found for update",
			slog.String("op", op),
			slog.String("session_id", sessionID.String()))
		return ErrNotFound
	}

	slog.Info("user session updated",
		slog.String("op", op),
		slog.String("session_id", sessionID.String()))
	return nil
}

// GetSessionByRefreshToken получает сессию по refresh токену
func (r *sessionRepository) GetSessionByRefreshToken(ctx context.Context, refreshToken string) (domain.Session, error) {
	const op = "postgres.sessionRepository.GetSessionByRefreshToken"

	query := `
		SELECT id, user_id, refresh_token, user_agent, 
		       ip_address::text, created_at, expires_at
		FROM user_sessions
		WHERE refresh_token = $1 AND expires_at > NOW()
	`

	session := domain.Session{}

	err := r.db.QueryRow(ctx, query, refreshToken).Scan(
		&session.ID,
		&session.UserID,
		&session.RefreshToken,
		&session.UserAgent,
		&session.IPAddress,
		&session.CreatedAt,
		&session.ExpiresAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Warn("session not found or expired",
				slog.String("op", op),
				slog.String("refresh_token_prefix", getTokenPrefix(refreshToken)))
			return domain.Session{}, ErrNotFound
		}

		slog.Error("failed to get session",
			slog.String("op", op),
			slog.Any("error", err))
		return domain.Session{}, fmt.Errorf("%s: %w", op, err)
	}

	return session, nil
}

// Delete удаляет сессию по ID
func (r *sessionRepository) Delete(ctx context.Context, sessionID uuid.UUID) error {
	const op = "postgres.sessionRepository.Delete"

	//// Конвертируем string в uuid.UUID
	//id, err := uuid.Parse(sessionID)
	//if err != nil {
	//	slog.Warn("invalid session ID format",
	//		slog.String("op", op),
	//		slog.String("session_id", sessionID))
	//	return fmt.Errorf("%s: invalid session ID", op)
	//}

	query := `DELETE FROM user_sessions WHERE id = $1`

	result, err := r.db.Exec(ctx, query, sessionID)
	if err != nil {
		slog.Error("failed to delete session",
			slog.String("op", op),
			slog.Any("session_id", sessionID),
			slog.Any("error", err))
		return fmt.Errorf("%s: %w", op, err)
	}

	if result.RowsAffected() == 0 {
		slog.Warn("session not found for deletion",
			slog.String("op", op),
			slog.Any("session_id", sessionID))
		return ErrNotFound
	}

	slog.Info("session deleted",
		slog.String("op", op),
		slog.Any("session_id", sessionID))

	return nil
}

// DeleteByUserID удаляет все сессии пользователя
func (r *sessionRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	const op = "postgres.sessionRepository.DeleteByUserID"

	// Конвертируем string в uuid.UUID
	//id, err := uuid.Parse(userID)
	//if err != nil {
	//	slog.Warn("invalid user ID format",
	//		slog.String("op", op),
	//		slog.String("user_id", userID))
	//	return fmt.Errorf("%s: invalid user ID", op)
	//}

	query := `DELETE FROM user_sessions WHERE user_id = $1`

	result, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		slog.Error("failed to delete user sessions",
			slog.String("op", op),
			slog.Any("user_id", userID),
			slog.Any("error", err))
		return fmt.Errorf("%s: %w", op, err)
	}

	slog.Info("all user sessions deleted",
		slog.String("op", op),
		slog.Any("user_id", userID),
		slog.Int64("deleted_count", result.RowsAffected()))

	return nil
}

// Вспомогательная функция для безопасного отображения токена
func getTokenPrefix(token string) string {
	if len(token) > 8 {
		return token[:8] + "..."
	}
	return token
}

// FindByID находит сессию по UUID
func (r *sessionRepository) FindByID(ctx context.Context, id uuid.UUID) (domain.Session, error) {
	const op = "postgres.sessionRepository.FindByID"

	query := `
		SELECT id, user_id, refresh_token, user_agent, 
		       ip_address::text, created_at, expires_at
		FROM user_sessions
		WHERE id = $1 AND expires_at > NOW()
	`

	session := domain.Session{}

	err := r.db.QueryRow(ctx, query, id).Scan(
		&session.ID,
		&session.UserID,
		&session.RefreshToken,
		&session.UserAgent,
		&session.IPAddress,
		&session.CreatedAt,
		&session.ExpiresAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Session{}, ErrNotFound
		}
		return domain.Session{}, fmt.Errorf("%s: %w", op, err)
	}

	return session, nil
}

// DeleteExpired удаляет истекшие сессии
func (r *sessionRepository) DeleteExpired(ctx context.Context) error {
	const op = "postgres.sessionRepository.DeleteExpired"

	query := `DELETE FROM user_sessions WHERE expires_at <= NOW()`
	result, err := r.db.Exec(ctx, query)

	if err != nil {
		slog.Error("failed to delete expired sessions",
			slog.String("op", op),
			slog.Any("error", err))
		return fmt.Errorf("%s: %w", op, err)
	}

	slog.Info("expired sessions deleted",
		"deleted_count", result.RowsAffected())
	return nil
}
