package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"
	"volunteer-coordination-server/internal/domain"
	"volunteer-coordination-server/internal/repository"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
}

// Register инициализируется ручками, вызывает [repository.QueryNewUser] и реализует добавление в бд нового пользователя
// Не поддерживает валидацию по ролям, она реализуется на уровне хендлеров
func (s *UserService) Register(ctx context.Context, reg domain.UserRegistration) (domain.User, error) {
	const op = "service.UserService.Register"

	// Проверяем, существует ли email
	exists, err := s.userRepo.ExistsByEmail(ctx, reg.Email)
	if err != nil {
		slog.Error("failed to check email existence",
			slog.String("op", op),
			slog.String("email", reg.Email),
			slog.Any("error", err))
		return domain.User{}, fmt.Errorf("%s: %w", op, err)
	}

	if exists {
		slog.Warn("email already registered",
			slog.String("op", op),
			slog.String("email", reg.Email))
		return domain.User{}, fmt.Errorf("%s: email already registered", op)
	}

	// Проверяем роли
	for _, role := range reg.Roles {
		if !domain.IsValidRole(role) {
			slog.Warn("invalid role in registration",
				slog.String("op", op),
				slog.String("email", reg.Email),
				slog.String("role", role))
			return domain.User{}, fmt.Errorf("%s: invalid role: %s", op, role)
		}
	}

	// Создаем доменный объект пользователя
	user, err := domain.NewUser(reg)
	if err != nil {
		slog.Error("failed to create user domain object",
			slog.String("op", op),
			slog.String("email", reg.Email),
			slog.Any("error", err))
		return domain.User{}, fmt.Errorf("%s: %w", op, err)
	}

	// Сохраняем пользователя в репозитории
	_, err = s.userRepo.Save(ctx, user)
	if err != nil {
		slog.Error("failed to save user",
			slog.String("op", op),
			slog.String("user_id", user.ID.String()),
			slog.String("email", user.Email),
			slog.Any("error", err))
		return domain.User{}, fmt.Errorf("%s: %w", op, err)
	}

	slog.Info("user registered successfully",
		slog.String("op", op),
		slog.String("user_id", user.ID.String()),
		slog.String("email", user.Email),
		slog.Int("roles_count", len(user.Roles)))

	return user, nil
}

// UserByParam принимает param - параметр по которому надо выполнить поиск (id или email) и value - значение параметра
// В зависимости от параметра вызывается нужная функция поиска в бд
func (s *UserService) UserByParam(ctx context.Context, param string, value string) (domain.User, error) {
	switch param {
	case "id":
		userID, err := uuid.Parse(value)
		if err != nil {
			return domain.User{}, fmt.Errorf("invalid UUID: %w", err)
		}

		usr, err := s.userRepo.QueryGetUserByID(ctx, userID)

		return usr, err
	case "email":
		usr, err := s.userRepo.QueryGetUserByEmail(ctx, value)
		return usr, err
	}
	return domain.User{}, nil
}

// UpdateUser обновляет данные пользователя
func (s *UserService) UpdateUser(ctx context.Context, user domain.User) (domain.User, error) {
	const op = "service.UserService.UpdateUser"

	existingUser, err := s.userRepo.QueryGetUserByID(ctx, user.ID)
	if err != nil {
		slog.Error("user not found for update",
			slog.String("op", op),
			slog.String("user_id", user.ID.String()))
		return domain.User{}, fmt.Errorf("%s: user not found", op)
	}

	if user.Email != existingUser.Email {
		exists, err := s.userRepo.ExistsByEmail(ctx, user.Email)
		if err != nil {
			slog.Error("failed to check email uniqueness",
				slog.String("op", op),
				slog.String("user_id", user.ID.String()),
				slog.String("new_email", user.Email),
				slog.Any("error", err))
			return domain.User{}, fmt.Errorf("%s: %w", op, err)
		}

		if exists {
			slog.Warn("email already in use",
				slog.String("op", op),
				slog.String("user_id", user.ID.String()),
				slog.String("new_email", user.Email))
			return domain.User{}, fmt.Errorf("%s: email already in use", op)
		}
	}

	for _, role := range user.Roles {
		if !domain.IsValidRole(role.Role) {
			slog.Warn("invalid role in update",
				slog.String("op", op),
				slog.String("user_id", user.ID.String()),
				slog.String("role", role.Role))
			return domain.User{}, fmt.Errorf("%s: invalid role: %s", op, role.Role)
		}
	}

	user.UpdatedAt = time.Now()

	// Сохраняем изменения
	err = s.userRepo.UpdateProfile(ctx, user)
	if err != nil {
		slog.Error("failed to update user",
			slog.String("op", op),
			slog.String("user_id", user.ID.String()),
			slog.Any("error", err))
		return domain.User{}, fmt.Errorf("%s: %w", op, err)
	}

	// Получаем обновленного пользователя
	updatedUser, err := s.userRepo.QueryGetUserByID(ctx, user.ID)
	if err != nil {
		slog.Error("failed to get updated user",
			slog.String("op", op),
			slog.String("user_id", user.ID.String()),
			slog.Any("error", err))
		return domain.User{}, fmt.Errorf("%s: %w", op, err)
	}

	slog.Info("user updated successfully",
		slog.String("op", op),
		slog.String("user_id", user.ID.String()))

	return updatedUser, nil
}

// ChangePassword меняет пароль пользователя
func (s *UserService) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	const op = "service.UserService.ChangePassword"

	id := userID

	user, err := s.userRepo.QueryGetUserByID(ctx, id)
	if err != nil {
		slog.Error("user not found for password change",
			slog.String("op", op),
			slog.String("user_id", userID.String()))
		return fmt.Errorf("%s: user not found", op)
	}

	err = user.CheckPassword(currentPassword)
	if err != nil {
		slog.Warn("invalid current password",
			slog.String("op", op),
			slog.String("user_id", user.ID.String()))
		return fmt.Errorf("%s: invalid current password", op)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("failed to hash new password",
			slog.String("op", op),
			slog.String("user_id", user.ID.String()),
			slog.Any("error", err))
		return fmt.Errorf("%s: %w", op, err)
	}

	user.Password = string(hashedPassword)
	user.UpdatedAt = time.Now()

	if err := s.userRepo.UpdatePassword(ctx, userID, string(hashedPassword)); err != nil {
		slog.Error("failed to update password",
			slog.String("op", op),
			slog.String("user_id", userID.String()),
			slog.Any("error", err))
		return fmt.Errorf("%s: %w", op, err)
	}

	slog.Info("password changed successfully",
		slog.String("op", op),
		slog.String("user_id", user.ID.String()))

	return nil
}

// UpdateProfile обновляет профиль пользователя
//func (s *UserService) UpdateProfile(ctx context.Context, userID, fullName, phone, bio string) (domain.User, error) {
//	const op = "service.UserService.UpdateProfile"
//
//	id, err := uuid.Parse(userID)
//	if err != nil {
//		slog.Warn("invalid UUID format",
//			slog.String("op", op),
//			slog.String("user_id", userID))
//		return domain.User{}, fmt.Errorf("%s: invalid UUID format", op)
//	}
//
//	// Получаем пользователя
//	user, err := s.userRepo.QueryGetUserByID(ctx, id)
//	if err != nil {
//		slog.Error("user not found for profile update",
//			slog.String("op", op),
//			slog.String("user_id", userID))
//		return domain.User{}, fmt.Errorf("%s: user not found", op)
//	}
//
//	// Обновляем поля
//	if fullName != "" {
//		user.FullName = fullName
//	}
//	if phone != "" {
//		user.Phone = phone
//	}
//	user.UpdatedAt = time.Now()
//
//	// Сохраняем изменения
//	err = s.userRepo.Save(ctx, user)
//	if err != nil {
//		slog.Error("failed to update profile",
//			slog.String("op", op),
//			slog.String("user_id", user.ID.String()),
//			slog.Any("error", err))
//		return domain.User{}, fmt.Errorf("%s: %w", op, err)
//	}
//
//	slog.Info("profile updated successfully",
//		slog.String("op", op),
//		slog.String("user_id", user.ID.String()))
//
//	return user, nil
//}

// BlockUser блокирует пользователя
func (s *UserService) BlockUser(ctx context.Context, userID string) error {
	const op = "service.UserService.BlockUser"

	id, err := uuid.Parse(userID)
	if err != nil {
		slog.Warn("invalid UUID format",
			slog.String("op", op),
			slog.String("user_id", userID))
		return fmt.Errorf("%s: invalid UUID format", op)
	}

	err = s.userRepo.BlockUser(ctx, id)
	if err != nil {
		slog.Error("failed to block user",
			slog.String("op", op),
			slog.String("user_id", userID),
			slog.Any("error", err))
		return fmt.Errorf("%s: %w", op, err)
	}

	slog.Info("user blocked successfully",
		slog.String("op", op),
		slog.String("user_id", userID))

	return nil
}

// UnblockUser разблокирует пользователя
func (s *UserService) UnblockUser(ctx context.Context, userID string) error {
	const op = "service.UserService.UnblockUser"

	id, err := uuid.Parse(userID)
	if err != nil {
		slog.Warn("invalid UUID format",
			slog.String("op", op),
			slog.String("user_id", userID))
		return fmt.Errorf("%s: invalid UUID format", op)
	}

	err = s.userRepo.UnblockUser(ctx, id)
	if err != nil {
		slog.Error("failed to unblock user",
			slog.String("op", op),
			slog.String("user_id", userID),
			slog.Any("error", err))
		return fmt.Errorf("%s: %w", op, err)
	}

	slog.Info("user unblocked successfully",
		slog.String("op", op),
		slog.String("user_id", userID))

	return nil
}

// AddUserRole добавляет роль пользователю
func (s *UserService) AddUserRole(ctx context.Context, userID, role string) error {
	const op = "service.UserService.AddUserRole"

	if !domain.IsValidRole(role) {
		slog.Warn("invalid role",
			slog.String("op", op),
			slog.String("role", role))
		return fmt.Errorf("%s: invalid role: %s", op, role)
	}

	id, err := uuid.Parse(userID)
	if err != nil {
		slog.Warn("invalid UUID format",
			slog.String("op", op),
			slog.String("user_id", userID))
		return fmt.Errorf("%s: invalid UUID format", op)
	}

	user, err := s.userRepo.QueryGetUserByID(ctx, id)
	if err != nil {
		slog.Error("user not found for adding role",
			slog.String("op", op),
			slog.String("user_id", userID))
		return fmt.Errorf("%s: user not found", op)
	}

	err = user.AddRole(role)
	if err != nil {
		slog.Error("failed to add role to user",
			slog.String("op", op),
			slog.String("user_id", user.ID.String()),
			slog.String("role", role),
			slog.Any("error", err))
		return fmt.Errorf("%s: %w", op, err)
	}

	err = s.userRepo.AddUserRole(ctx, user.ID, role)
	if err != nil {
		slog.Error("failed to save user with new role",
			slog.String("op", op),
			slog.String("user_id", user.ID.String()),
			slog.Any("error", err))
		return fmt.Errorf("%s: %w", op, err)
	}

	slog.Info("role added to user successfully",
		slog.String("op", op),
		slog.String("user_id", user.ID.String()),
		slog.String("role", role))

	return nil
}

// RemoveUserRole удаляет роль у пользователя
func (s *UserService) RemoveUserRole(ctx context.Context, userID, role string) error {
	const op = "service.UserService.RemoveUserRole"

	id, err := uuid.Parse(userID)
	if err != nil {
		slog.Warn("invalid UUID format",
			slog.String("op", op),
			slog.String("user_id", userID))
		return fmt.Errorf("%s: invalid UUID format", op)
	}

	// Получаем пользователя
	user, err := s.userRepo.QueryGetUserByID(ctx, id)
	if err != nil {
		slog.Error("user not found for removing role",
			slog.String("op", op),
			slog.String("user_id", userID))
		return fmt.Errorf("%s: user not found", op)
	}

	// Удаляем роль
	removed := user.RemoveRole(role)
	if !removed {
		slog.Warn("role not found on user",
			slog.String("op", op),
			slog.String("user_id", user.ID.String()),
			slog.String("role", role))
		return fmt.Errorf("%s: role not found on user", op)
	}

	// Сохраняем изменения
	_, err = s.userRepo.Save(ctx, user)
	if err != nil {
		slog.Error("failed to save user after removing role",
			slog.String("op", op),
			slog.String("user_id", user.ID.String()),
			slog.Any("error", err))
		return fmt.Errorf("%s: %w", op, err)
	}

	slog.Info("role removed from user successfully",
		slog.String("op", op),
		slog.String("user_id", user.ID.String()),
		slog.String("role", role))

	return nil
}

// GetUsers возвращает список пользователей с пагинацией
// Пока заглушка
func (s *UserService) GetUsers(ctx context.Context, pageStr, limitStr, roleFilter string) ([]*domain.User, int64, error) {
	const op = "service.UserService.GetUsers"

	return []*domain.User{}, 0, fmt.Errorf("%s: not implemented", op)
}
