package mailer

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"strings"
)

// Sender delivers one plain-text email. Every call site treats a non-nil
// error as "the recipient did NOT get this" -- CAP-O10's own invitation
// handler surfaces it back to the inviting Admin rather than pretending
// success, same "don't lie about what happened" discipline this codebase
// already applies to Reload (admin.go).
type Sender interface {
	Send(ctx context.Context, to, subject, body string) error
}

// Config is the SMTP transport's own connection details -- see
// config.Config's SMTP* fields, loaded from plain env vars per this
// package's doc comment.
type Config struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

// New returns a real SMTPSender when Host is set, or a log-only Sender
// otherwise (local dev, CI, or a deployment that hasn't configured a relay
// yet) -- see doc.go for why this is a deliberate fallback, not a silently
// swallowed error.
func New(cfg Config) Sender {
	if cfg.Host == "" {
		return logSender{}
	}
	return smtpSender{cfg: cfg}
}

type logSender struct{}

func (logSender) Send(_ context.Context, to, subject, body string) error {
	slog.Info("email not sent (SMTP_HOST not configured)", "to", to, "subject", subject, "body", body)
	return nil
}

type smtpSender struct {
	cfg Config
}

// Send dials the configured relay and delivers one message. Supports both
// implicit TLS (port 465 -- dial straight into TLS, no STARTTLS handshake)
// and STARTTLS (every other port, typically 587) -- the two shapes real
// relays actually offer; smtp.SendMail alone only ever does the latter.
func (s smtpSender) Send(_ context.Context, to, subject, body string) error {
	addr := net.JoinHostPort(s.cfg.Host, s.cfg.Port)
	auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	msg := buildMessage(s.cfg.From, to, subject, body)

	if s.cfg.Port == "465" {
		conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: s.cfg.Host})
		if err != nil {
			return fmt.Errorf("dial smtp (tls): %w", err)
		}
		defer conn.Close()
		client, err := smtp.NewClient(conn, s.cfg.Host)
		if err != nil {
			return fmt.Errorf("smtp handshake: %w", err)
		}
		defer client.Close()
		return sendVia(client, auth, s.cfg.From, to, msg)
	}

	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("dial smtp: %w", err)
	}
	defer client.Close()
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: s.cfg.Host}); err != nil {
			return fmt.Errorf("starttls: %w", err)
		}
	}
	return sendVia(client, auth, s.cfg.From, to, msg)
}

func sendVia(client *smtp.Client, auth smtp.Auth, from, to string, msg []byte) error {
	if ok, _ := client.Extension("AUTH"); ok {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close: %w", err)
	}
	return client.Quit()
}

func buildMessage(from, to, subject, body string) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", from)
	fmt.Fprintf(&b, "To: %s\r\n", to)
	fmt.Fprintf(&b, "Subject: %s\r\n", subject)
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	return []byte(b.String())
}
