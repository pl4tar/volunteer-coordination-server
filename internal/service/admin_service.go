package service

import (
	"context"
	"errors"
	"fmt"
	"volunteer-coordination-server/internal/domain"

	"volunteer-coordination-server/internal/repository"

	"github.com/google/uuid"
)

type AdminService struct {
	users  repository.UserRepository
	mod    repository.ModerationRepository
	cats   repository.CategoryRepository
	tags   repository.TagRepository
	stats  repository.StatsRepository
	setts  repository.SettingsRepository
	verify repository.OrganizerVerificationRepository
}

func NewAdminService(
	users repository.UserRepository,
	mod repository.ModerationRepository,
	cats repository.CategoryRepository,
	tags repository.TagRepository,
	stats repository.StatsRepository,
	setts repository.SettingsRepository,
	verify repository.OrganizerVerificationRepository,
) *AdminService {
	return &AdminService{users: users, mod: mod, cats: cats, tags: tags, stats: stats, setts: setts, verify: verify}
}

// --- users ---
func (s *AdminService) BlockUser(ctx context.Context, userID uuid.UUID) error {
	return s.users.BlockUser(ctx, userID)
}
func (s *AdminService) UnblockUser(ctx context.Context, userID uuid.UUID) error {
	return s.users.UnblockUser(ctx, userID)
}

// --- moderation ---
func (s *AdminService) ModerateEvent(ctx context.Context, adminID, eventID uuid.UUID, status string, note string) error {
	switch status {
	case "active", "rejected", "pending_moderation", "cancelled", "finished":
	default:
		return fmt.Errorf("invalid status")
	}
	return s.mod.SetEventStatus(ctx, adminID, eventID, status, map[string]any{"note": note})
}

// --- categories ---
func (s *AdminService) ListCategories(ctx context.Context) ([]domain.Category, error) {
	return s.cats.List(ctx)
}
func (s *AdminService) CreateCategory(ctx context.Context, name, slug, description string) (int, error) {
	if name == "" || slug == "" {
		return 0, errors.New("name and slug required")
	}
	return s.cats.Create(ctx, name, slug, description)
}
func (s *AdminService) DeleteCategory(ctx context.Context, id int) error {
	return s.cats.Delete(ctx, id)
}

// --- tags ---
func (s *AdminService) ListTags(ctx context.Context) ([]domain.Tag, error) {
	return s.tags.List(ctx)
}
func (s *AdminService) CreateTag(ctx context.Context, name string) (int, error) {
	if name == "" {
		return 0, errors.New("name required")
	}
	return s.tags.Create(ctx, name)
}
func (s *AdminService) DeleteTag(ctx context.Context, id int) error { return s.tags.Delete(ctx, id) }

// --- stats ---
func (s *AdminService) SystemStats(ctx context.Context) (map[string]any, error) {
	return s.stats.SystemStats(ctx)
}

// --- settings ---
func (s *AdminService) GetSettings(ctx context.Context, key string) (map[string]any, error) {
	return s.setts.Get(ctx, key)
}
func (s *AdminService) SetSettings(ctx context.Context, key string, val map[string]any) error {
	if key == "" {
		return errors.New("key required")
	}
	return s.setts.Set(ctx, key, val)
}

// --- organizer verification ---
// ВАЖНО: этот метод используется в OrganizerHandler (эндпоинт /me/organizer/request)
func (s *AdminService) RequestOrganizer(ctx context.Context, userID uuid.UUID, companyName string, documents map[string]any) error {
	return s.verify.Request(ctx, userID, companyName, documents)
}

func (s *AdminService) ListOrganizerVerifications(ctx context.Context, status string, page, limit int) ([]domain.OrganizerVerification, int, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit
	return s.verify.List(ctx, status, limit, offset)
}

func (s *AdminService) DecideOrganizerVerification(ctx context.Context, adminID, userID uuid.UUID, approve bool, note string) error {
	return s.verify.Decide(ctx, adminID, userID, approve, note)
}
