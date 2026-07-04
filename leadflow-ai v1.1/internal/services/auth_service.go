package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/leadflow-ai/leadflow-ai/internal/auth"
	"github.com/leadflow-ai/leadflow-ai/internal/models"
	"github.com/leadflow-ai/leadflow-ai/internal/repositories"
)

var (
	// ErrEmailTaken пробрасывается наружу из репозитория как есть, чтобы
	// обработчик мог вернуть 409 Conflict, не заглядывая внутрь пакета
	// repositories.
	ErrEmailTaken = repositories.ErrEmailTaken

	ErrInvalidCredentials = errors.New("auth_service: invalid email or password")
)

// AuthService инкапсулирует всю бизнес-логику аутентификации: валидацию
// входных данных, хэширование паролей и выпуск токенов. Обработчик HTTP
// (api.AuthHandler) не знает ни про bcrypt/pbkdf2, ни про формат JWT —
// он лишь вызывает Register/Login и сериализует результат.
type AuthService struct {
	users  *repositories.UserRepository
	tokens *auth.TokenIssuer
}

func NewAuthService(users *repositories.UserRepository, tokens *auth.TokenIssuer) *AuthService {
	return &AuthService{users: users, tokens: tokens}
}

func (s *AuthService) Register(ctx context.Context, email, password, name string) (*models.User, string, error) {
	email = normalizeEmail(email)
	name = strings.TrimSpace(name)

	if !isValidEmail(email) {
		return nil, "", fmt.Errorf("auth_service: invalid email address")
	}
	if name == "" {
		return nil, "", fmt.Errorf("auth_service: name must not be empty")
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return nil, "", fmt.Errorf("auth_service: %w", err)
	}

	user := &models.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: passwordHash,
		Name:         name,
		Role:         models.UserRoleMember,
	}

	if err := s.users.Create(ctx, user); err != nil {
		if errors.Is(err, repositories.ErrEmailTaken) {
			return nil, "", ErrEmailTaken
		}
		return nil, "", fmt.Errorf("auth_service: register: %w", err)
	}

	token, err := s.tokens.Issue(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, "", fmt.Errorf("auth_service: issue token: %w", err)
	}

	return user, token, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*models.User, string, error) {
	email = normalizeEmail(email)

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, "", ErrInvalidCredentials
	}

	if err := auth.VerifyPassword(user.PasswordHash, password); err != nil {
		return nil, "", ErrInvalidCredentials
	}

	token, err := s.tokens.Issue(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, "", fmt.Errorf("auth_service: issue token: %w", err)
	}

	return user, token, nil
}

func (s *AuthService) Profile(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("auth_service: profile: %w", err)
	}
	return user, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// isValidEmail — минимальная синтаксическая проверка, а не полная
// валидация по RFC 5322. Единственная авторитетная проверка того, что
// email реально существует и принадлежит пользователю, — это письмо
// с подтверждением, которого в этом инкременте намеренно нет; здесь
// цель лишь отсеять явный мусор до похода в БД.
func isValidEmail(email string) bool {
	at := strings.IndexByte(email, '@')
	if at <= 0 || at == len(email)-1 {
		return false
	}
	domain := email[at+1:]
	return strings.Contains(domain, ".") && !strings.HasSuffix(domain, ".")
}
