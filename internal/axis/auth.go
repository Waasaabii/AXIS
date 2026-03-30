package axis

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/scrypt"
)

const hashPrefix = "scrypt"

type AuthContext struct {
	username   string
	password   string
	passwordHash string
	secret     string
	ttl        time.Duration
}

type sessionPayload struct {
	Username string `json:"username"`
	Exp      int64  `json:"exp"`
	Nonce    string `json:"nonce"`
}

func randomHex(size int) string {
	buf := make([]byte, size)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

func base64URLEncode(input []byte) string {
	return base64.RawURLEncoding.EncodeToString(input)
}

func base64URLDecode(input string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(input)
}

func constantTimeEqualString(left, right string) bool {
	if len(left) != len(right) {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}

func CreatePasswordHash(password string) (string, error) {
	salt := randomHex(16)
	derived, err := scrypt.Key([]byte(password), []byte(salt), 16384, 8, 1, 64)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s$%s$%s", hashPrefix, salt, hex.EncodeToString(derived)), nil
}

func verifyPasswordHash(password, encodedHash string) bool {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 3 || parts[0] != hashPrefix || parts[1] == "" || parts[2] == "" {
		return false
	}

	actual, err := scrypt.Key([]byte(password), []byte(parts[1]), 16384, 8, 1, 64)
	if err != nil {
		return false
	}

	expected, err := hex.DecodeString(parts[2])
	if err != nil {
		return false
	}

	if len(actual) != len(expected) {
		return false
	}

	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func ParseCookies(cookieHeader string) map[string]string {
	cookies := map[string]string{}
	parts := strings.Split(cookieHeader, ";")
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		sections := strings.SplitN(item, "=", 2)
		if len(sections) != 2 {
			continue
		}
		cookies[sections[0]] = sections[1]
	}
	return cookies
}

func NewAuthContext(username, passwordHash, password, sessionSecret string, sessionTTLHours int) *AuthContext {
	ttl := time.Duration(max(1, sessionTTLHours)) * time.Hour
	secret := sessionSecret
	if secret == "" {
		sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%s", username, firstNonEmpty(passwordHash, password, "proxyrelay"))))
		secret = hex.EncodeToString(sum[:])
	}

	return &AuthContext{
		username: username,
		passwordHash: passwordHash,
		password: password,
		secret: secret,
		ttl: ttl,
	}
}

func (a *AuthContext) VerifyCredentials(username, password string) bool {
	if username != a.username {
		return false
	}

	if a.passwordHash != "" {
		return verifyPasswordHash(password, a.passwordHash)
	}

	if a.password != "" {
		return constantTimeEqualString(password, a.password)
	}

	return false
}

func (a *AuthContext) sign(encodedPayload string) string {
	mac := hmac.New(sha256.New, []byte(a.secret))
	mac.Write([]byte(encodedPayload))
	return hex.EncodeToString(mac.Sum(nil))
}

func (a *AuthContext) CreateSessionCookie(sessionUsername string) (string, error) {
	payload := sessionPayload{
		Username: sessionUsername,
		Exp:      time.Now().Add(a.ttl).UnixMilli(),
		Nonce:    randomHex(12),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	encoded := base64URLEncode(raw)
	signature := a.sign(encoded)
	return fmt.Sprintf("proxyrelay_session=%s.%s; Path=/; HttpOnly; SameSite=Strict; Max-Age=%d", encoded, signature, int(a.ttl.Seconds())), nil
}

func (a *AuthContext) ClearSessionCookie() string {
	return "proxyrelay_session=; Path=/; HttpOnly; SameSite=Strict; Max-Age=0"
}

func (a *AuthContext) ReadSession(cookieValue string) (bool, string) {
	parts := strings.SplitN(cookieValue, ".", 2)
	if len(parts) != 2 {
		return false, ""
	}

	if !constantTimeEqualString(parts[1], a.sign(parts[0])) {
		return false, ""
	}

	decoded, err := base64URLDecode(parts[0])
	if err != nil {
		return false, ""
	}

	var payload sessionPayload
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return false, ""
	}

	if payload.Exp < time.Now().UnixMilli() {
		return false, ""
	}

	return true, payload.Username
}

func writeSetCookie(w http.ResponseWriter, value string) {
	w.Header().Set("Set-Cookie", value)
}
