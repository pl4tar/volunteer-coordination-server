package service

import (
	"context"
	"log/slog"
	"volunteer-coordination-server/internal/entity"
	"volunteer-coordination-server/internal/repository/UserRepository"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserService struct {
	userRepo *UserRepository.UsrRepository
}

func NewUserService(db *pgxpool.Pool) *UserService {
	return &UserService{
		userRepo: UserRepository.NewUsrRepository(db),
	}
}

// AddNewUser инициализируется ручками, вызывает QueryNewUser и реализует добавление в бд нового пользователя
// Не поддерживает валидацию по ролям, она реализуется на уровне хендлеров
func (s *UserService) AddNewUser(ctx context.Context, usr entity.UserRegistration) (string, error) {
	id, err := s.userRepo.QueryNewUser(ctx, usr)
	if err != nil {
		slog.Error("Query error:", err.Error())
		return "", err
	}
	return id, nil
}

// UserByParam принимает param - параметр по которому надо выполнить поиск (id или email) и value - значение параметра
// В зависимости от параметра вызывается нужная функция поиска в бд
func (s *UserService) UserByParam(ctx context.Context, param string, value string) (entity.User, error) {
	switch param {
	case "id":
		usr, err := s.userRepo.QueryGetUserByID(ctx, value)
		return usr, err
	case "email":
		usr, err := s.userRepo.QueryGetUserByEmail(ctx, value)
		return usr, err
	}
	return entity.User{}, nil
}
