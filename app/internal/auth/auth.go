package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// SessionTTL is how long a session stays valid without activity -- sliding,
// refreshed on every authenticated request (cmd/server/main.go's sessionAuth
// middleware Touch call) and used to set the session cookie's Expires at
// login (internal/handler's Login).
const SessionTTL = 24 * time.Hour

// HashPassword bcrypt-hashes a plaintext password for storage
// (users.password_hash). Cost 10, matching this prototype's existing seed
// convention (Case 1's Alice/Bob rows).
func HashPassword(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword reports whether plain matches the bcrypt hash. false on any
// error (malformed hash, mismatch) -- a verification failure is always just
// "not authenticated," never a reason to leak why.
func VerifyPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// NewToken generates a fresh 32-byte random bearer token (session cookie
// value or CSRF token), base64url-encoded. Used for both session tokens and
// CSRF tokens -- same entropy requirement, same primitive.
func NewToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashSessionToken returns the value actually stored as sessions.id --
// SHA-256(token), hex. The raw token only ever lives in the client's cookie
// and in-memory during a request; a leaked database row alone can't be
// replayed as a session, the same "don't store the secret itself" principle
// password_hash already applies to passwords.
func HashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// ConstantTimeEqual compares two tokens (CSRF token, session lookups) in
// time independent of where they first differ -- guards against a timing
// side-channel letting an attacker guess a valid token byte by byte.
func ConstantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
