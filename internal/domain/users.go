package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidEmail     = errors.New("invalid email")
	ErrPasswordTooShort = errors.New("password must be at least 6 characters")
	ErrInvalidRole      = errors.New("invalid role")
	ErrUserBlocked      = errors.New("user is blocked")
	ErrInvalidPassword  = errors.New("invalid password")
	ErrSessionExpired   = errors.New("session expired")
)

// UserRole - структура для хранения роли пользователя с временем выдачи
type UserRole struct {
	Role      string    `json:"role" db:"role"`
	GrantedAt time.Time `json:"granted_at" db:"granted_at"`
}

type User struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	Email        string     `json:"email" db:"email"`
	PasswordHash string     `json:"-" db:"password_hash"` // Скрываем в JSON
	FullName     string     `json:"full_name" db:"full_name"`
	Phone        string     `json:"phone,omitempty" db:"phone"`
	Roles        []UserRole `json:"roles,omitempty" db:"-"` // Не маппим напрямую, роли получаем отдельно
	IsBlocked    bool       `json:"is_blocked,omitempty" db:"is_blocked"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

type UserRegistration struct {
	Email    string   `json:"email" binding:"required,email"`
	Password string   `json:"password" binding:"required,min=6"`
	FullName string   `json:"full_name" binding:"required"`
	Phone    string   `json:"phone,omitempty"`
	Roles    []string `json:"roles,omitempty"` // Принимаем массив строк для ролей
}

func NewUser(reg UserRegistration) (*User, error) {
	if reg.Email == "" {
		return nil, ErrInvalidEmail
	}

	if len(reg.Password) < 6 {
		return nil, ErrPasswordTooShort
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(reg.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	user := &User{
		ID:           uuid.New(),
		Email:        reg.Email,
		PasswordHash: string(hashedPassword),
		FullName:     reg.FullName,
		Phone:        reg.Phone,
		IsBlocked:    false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Добавляем роли (если они указаны)
	if len(reg.Roles) > 0 {
		for _, role := range reg.Roles {
			if err := user.AddRole(role); err != nil {
				return nil, err
			}
		}
	} else {
		// По умолчанию добавляем роль volunteer
		if err := user.AddRole("volunteer"); err != nil {
			return nil, err
		}
	}

	return user, nil
}

// AddRole добавляет роль пользователю
func (u *User) AddRole(role string) error {
	if !isValidRole(role) {
		return ErrInvalidRole
	}

	// Проверяем, есть ли уже такая роль
	for _, r := range u.Roles {
		if r.Role == role {
			return nil // Роль уже есть
		}
	}

	u.Roles = append(u.Roles, UserRole{
		Role:      role,
		GrantedAt: time.Now(),
	})

	return nil
}

// RemoveRole удаляет роль у пользователя
func (u *User) RemoveRole(role string) bool {
	for i, r := range u.Roles {
		if r.Role == role {
			u.Roles = append(u.Roles[:i], u.Roles[i+1:]...)
			return true
		}
	}
	return false
}

// GetRoles возвращает список ролей как строки
func (u *User) GetRoles() []string {
	roles := make([]string, len(u.Roles))
	for i, r := range u.Roles {
		roles[i] = r.Role
	}
	return roles
}

// GetRoleStrings возвращает строки ролей (удобно для JWT токенов)
func (u *User) GetRoleStrings() []string {
	return u.GetRoles()
}

// HasRole проверяет наличие роли
func (u *User) HasRole(role string) bool {
	for _, r := range u.Roles {
		if r.Role == role {
			return true
		}
	}
	return false
}

// HasAnyRole проверяет наличие любой из указанных ролей
func (u *User) HasAnyRole(roles ...string) bool {
	for _, userRole := range u.Roles {
		for _, checkRole := range roles {
			if userRole.Role == checkRole {
				return true
			}
		}
	}
	return false
}

// CheckPassword проверяет пароль
func (u *User) CheckPassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
}

// Block блокирует пользователя
func (u *User) Block() {
	u.IsBlocked = true
	u.UpdatedAt = time.Now()
}

// Unblock разблокирует пользователя
func (u *User) Unblock() {
	u.IsBlocked = false
	u.UpdatedAt = time.Now()
}

// UpdateProfile обновляет профиль пользователя
func (u *User) UpdateProfile(fullName, phone string) {
	if fullName != "" {
		u.FullName = fullName
	}
	if phone != "" {
		u.Phone = phone
	}
	u.UpdatedAt = time.Now()
}

// ChangePassword меняет пароль
func (u *User) ChangePassword(newPassword string) error {
	if len(newPassword) < 6 {
		return ErrPasswordTooShort
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	u.PasswordHash = string(hashedPassword)
	u.UpdatedAt = time.Now()
	return nil
}

func isValidRole(role string) bool {
	validRoles := map[string]bool{
		"volunteer": true,
		"organizer": true,
		"admin":     true,
	}
	return validRoles[role]
}

// IsAdmin проверяет, является ли пользователь администратором
func (u *User) IsAdmin() bool {
	return u.HasRole("admin")
}

// IsOrganizer проверяет, является ли пользователь организатором
func (u *User) IsOrganizer() bool {
	return u.HasRole("organizer")
}

// IsVolunteer проверяет, является ли пользователь волонтером
func (u *User) IsVolunteer() bool {
	return u.HasRole("volunteer")
}

// ToPublicDTO возвращает публичное представление пользователя (без пароля)
func (u *User) ToPublicDTO() map[string]interface{} {
	return map[string]interface{}{
		"id":         u.ID,
		"email":      u.Email,
		"full_name":  u.FullName,
		"phone":      u.Phone,
		"roles":      u.GetRoles(),
		"is_blocked": u.IsBlocked,
		"created_at": u.CreatedAt,
		"updated_at": u.UpdatedAt,
	}
}
