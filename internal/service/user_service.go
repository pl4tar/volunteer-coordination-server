package service

import (
	"context"
	"fmt"
	"log/slog"
	"volunteer-coordination-server/internal/domain"
	"volunteer-coordination-server/internal/repository"

	"github.com/google/uuid"
)

type UserService struct {
	userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
}

// Register инициализируется ручками, вызывает QueryNewUser и реализует добавление в бд нового пользователя
// Не поддерживает валидацию по ролям, она реализуется на уровне хендлеров
func (s *UserService) Register(ctx context.Context, usr domain.UserRegistration) (string, error) {
	id, err := s.userRepo.QueryNewUser(ctx, usr)
	if err != nil {
		slog.Error("Query error:", err.Error())
		return "", err
	}
	return id, nil
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
