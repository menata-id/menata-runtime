package config

import "os"

type Config struct {
	DatabaseURL string
	Port        string
	// SecureCookies (CAP-X02) sets the session cookie's Secure attribute --
	// must be true in production (menata.app is HTTPS-only), but a local
	// dev server over plain http://localhost needs it false or browsers
	// refuse to store the cookie at all, silently breaking login. Defaults
	// false (dev-friendly); production's .env sets SECURE_COOKIES=true.
	SecureCookies bool
	// SMTP* (CAP-O10) is the one genuinely new infrastructure dependency
	// Study 35 §5.6 named and deliberately left unsolved: this runtime has
	// never sent a real outbound email before. Plain SMTP (not a vendor
	// API/SDK) per the same "Infer Before Configure" reasoning Phase 3
	// already applied to secrets -- no deployment target here forces a
	// specific transactional-email vendor, and SMTP is the one transport
	// every relay (a real mailbox, Postmark, SES, Mailgun...) speaks, so
	// picking it doesn't foreclose swapping providers later, only how they're
	// reached. SMTPHost empty (the dev-friendly default) makes
	// internal/mailer.New return a log-only Sender instead of a real one --
	// see that package's own doc comment.
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string
}

func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3100"
	}
	smtpPort := os.Getenv("SMTP_PORT")
	if smtpPort == "" {
		smtpPort = "587"
	}
	return Config{
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		Port:          port,
		SecureCookies: os.Getenv("SECURE_COOKIES") == "true",
		SMTPHost:      os.Getenv("SMTP_HOST"),
		SMTPPort:      smtpPort,
		SMTPUsername:  os.Getenv("SMTP_USERNAME"),
		SMTPPassword:  os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:      os.Getenv("SMTP_FROM"),
	}
}
