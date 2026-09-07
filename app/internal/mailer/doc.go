// Package mailer sends real outbound email -- the one genuinely new
// infrastructure dependency CAP-O10 (workspace invitations) needs, named
// but deliberately left unsolved by Study 35 §5.6
// (../../../benchmarks/027-workspace-self-service-provisioning-study.md):
// this runtime had never sent an email before this package existed.
//
// NEW package, not a graduated one. Transport: plain SMTP (net/smtp,
// stdlib only, no new go.mod dependency), not a transactional-email vendor
// SDK (Postmark/SES/etc.) -- per this project's own "Infer Before
// Configure" principle, the same reasoning app/ROADMAP.md's Phase 3 already
// applied to secrets. No case here forces a specific vendor, and every
// relay worth using (a real mailbox, Postmark, SES, Mailgun, ...) speaks
// SMTP, so this choice doesn't foreclose switching providers later, only
// how one is reached -- reversible at the config layer, not the code layer.
//
// config.Config.SMTPHost empty (the dev-friendly default, and every CI run)
// makes New return a logSender instead of a real SMTPSender: outbound mail
// is logged at info level, not silently dropped, so nothing breaks and
// nothing lies about having sent something it didn't. Set SMTP_HOST (plus
// SMTP_PORT/SMTP_USERNAME/SMTP_PASSWORD/SMTP_FROM) once a real relay is
// available.
package mailer
