package config

import "os"

type Config struct {
	DatabaseURL string
	Port        string
	// SecureCookies (CAP-X02) sets the session cookie's Secure attribute --
	// must be true in production (aksi.menata.id is HTTPS-only), but a local
	// dev server over plain http://localhost needs it false or browsers
	// refuse to store the cookie at all, silently breaking login. Defaults
	// false (dev-friendly); production's .env sets SECURE_COOKIES=true.
	SecureCookies bool
}

func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3100"
	}
	return Config{
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		Port:          port,
		SecureCookies: os.Getenv("SECURE_COOKIES") == "true",
	}
}
