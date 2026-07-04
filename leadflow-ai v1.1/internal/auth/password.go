package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

// pbkdf2Iterations — число итераций HMAC-SHA256. 210_000 соответствует
// текущей (2023+) рекомендации OWASP для PBKDF2-HMAC-SHA256 и даёт
// хэширование одного пароля порядка десятков миллисекунд — достаточно
// медленно против перебора, но не заметно пользователю при логине.
const pbkdf2Iterations = 210_000

const saltBytes = 16

// HashPassword возвращает самоописывающуюся строку вида
// "pbkdf2-sha256$<iterations>$<saltBase64>$<hashBase64>". Формат хранит
// параметры хэширования внутри самой строки, чтобы итерации можно было
// увеличить в будущем без миграции уже сохранённых хэшей: старые хэши
// продолжат проверяться с тем количеством итераций, с которым были
// созданы.
func HashPassword(password string) (string, error) {
	if len(password) < 8 {
		return "", fmt.Errorf("auth: password must be at least 8 characters")
	}

	salt := make([]byte, saltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("auth: generate salt: %w", err)
	}

	hash := pbkdf2HMACSHA256(password, salt, pbkdf2Iterations)

	encoded := fmt.Sprintf("pbkdf2-sha256$%d$%s$%s",
		pbkdf2Iterations,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)
	return encoded, nil
}

// VerifyPassword сверяет пароль с ранее сохранённым хэшем. Сравнение
// хэшей выполняется через subtle.ConstantTimeCompare, чтобы время
// проверки не зависело от того, на каком байте хэши разошлись
// (защита от timing-атак при подборе пароля).
func VerifyPassword(encodedHash, password string) error {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" {
		return fmt.Errorf("auth: unrecognized password hash format")
	}

	iterations, err := strconv.Atoi(parts[1])
	if err != nil || iterations <= 0 {
		return fmt.Errorf("auth: invalid iteration count in password hash")
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return fmt.Errorf("auth: invalid salt encoding in password hash")
	}

	expected, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return fmt.Errorf("auth: invalid hash encoding in password hash")
	}

	actual := pbkdf2HMACSHA256(password, salt, iterations)
	if subtle.ConstantTimeCompare(actual, expected) != 1 {
		return fmt.Errorf("auth: password does not match")
	}
	return nil
}

// pbkdf2HMACSHA256 — минимальная реализация PBKDF2 (RFC 8018) на базе
// crypto/hmac и crypto/sha256 из стандартной библиотеки. Возвращает ключ
// длиной sha256.Size (32 байта), что достаточно для использования как
// пароль-хэш.
func pbkdf2HMACSHA256(password string, salt []byte, iterations int) []byte {
	mac := hmac.New(sha256.New, []byte(password))
	dkLen := sha256.Size

	var block uint32 = 1
	result := make([]byte, 0, dkLen)

	for len(result) < dkLen {
		mac.Reset()
		mac.Write(salt)
		mac.Write([]byte{
			byte(block >> 24),
			byte(block >> 16),
			byte(block >> 8),
			byte(block),
		})
		u := mac.Sum(nil)

		t := make([]byte, len(u))
		copy(t, u)

		for i := 1; i < iterations; i++ {
			mac.Reset()
			mac.Write(u)
			u = mac.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}

		result = append(result, t...)
		block++
	}

	return result[:dkLen]
}
