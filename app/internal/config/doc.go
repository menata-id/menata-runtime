// Package config loads runtime configuration (DatabaseURL, Port,
// SecureCookies, ...) into a Config struct.
//
// Graduated as-is from prototype/go/internal/config — a leaf package, 0
// internal/ dependencies (Study 34 §7 Method 2). One deferred change, named
// not solved here: Study 34 §5 found secrets (DATABASE_URL's own password)
// read from a plaintext .env file, with no vault/secrets-manager
// integration anywhere. This package is the one integration point a real
// secrets backend would plug into later — plain env-var loading stays the
// default until a real requirement forces the change, per this project's
// own "Infer Before Configure" principle.
package config
