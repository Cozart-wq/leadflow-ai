package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// header — фиксированный JWT-заголовок. Алгоритм не выбирается динамически:
// TokenIssuer поддерживает единственный алгоритм (HS256), поэтому здесь нет
// смысла в общем JWT-парсере, принимающем произвольный alg, — это лишь
// расширяет поверхность атаки (classic "alg: none" и confusion-атаки на
// JWT-библиотеки).
var jwtHeader = base64URLEncode([]byte(`{"alg":"HS256","typ":"JWT"}`))

// Claims — набор данных, зашитых в токен. UserID используется как ключ
// авторизации на всех защищённых маршрутах; Email и Role дублируются в
// токене, чтобы обработчикам не требовался поход в БД только ради
// отображения имени пользователя или проверки роли.
type Claims struct {
	UserID    uuid.UUID `json:"uid"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	IssuedAt  int64     `json:"iat"`
	ExpiresAt int64     `json:"exp"`
}

func (c Claims) expired(now time.Time) bool {
	return now.Unix() >= c.ExpiresAt
}

// TokenIssuer выпускает и проверяет JWT, подписанные общим секретом
// (HS256). Секрет передаётся в конструктор, а не читается из глобального
// состояния — это позволяет тестировать выпуск/проверку токенов без
// поднятия конфигурации всего приложения.
type TokenIssuer struct {
	secret []byte
	ttl    time.Duration
}

func NewTokenIssuer(secret string, ttl time.Duration) (*TokenIssuer, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("auth: jwt secret must be at least 32 characters")
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("auth: token ttl must be positive")
	}
	return &TokenIssuer{secret: []byte(secret), ttl: ttl}, nil
}

// Issue выпускает подписанный access-токен для пользователя.
func (i *TokenIssuer) Issue(userID uuid.UUID, email, role string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:    userID,
		Email:     email,
		Role:      role,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(i.ttl).Unix(),
	}

	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("auth: marshal claims: %w", err)
	}

	encodedPayload := base64URLEncode(payload)
	signingInput := jwtHeader + "." + encodedPayload
	signature := i.sign(signingInput)

	return signingInput + "." + signature, nil
}

// Parse проверяет подпись и срок действия токена и возвращает claims.
func (i *TokenIssuer) Parse(token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("auth: malformed token")
	}

	signingInput := parts[0] + "." + parts[1]
	expectedSignature := i.sign(signingInput)

	if subtle.ConstantTimeCompare([]byte(parts[2]), []byte(expectedSignature)) != 1 {
		return nil, fmt.Errorf("auth: invalid token signature")
	}

	payload, err := base64URLDecode(parts[1])
	if err != nil {
		return nil, fmt.Errorf("auth: invalid token payload encoding: %w", err)
	}

	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("auth: invalid token payload: %w", err)
	}

	if claims.expired(time.Now()) {
		return nil, fmt.Errorf("auth: token expired")
	}

	return &claims, nil
}

func (i *TokenIssuer) sign(signingInput string) string {
	mac := hmac.New(sha256.New, i.secret)
	mac.Write([]byte(signingInput))
	return base64URLEncode(mac.Sum(nil))
}

func base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func base64URLDecode(data string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(data)
}
