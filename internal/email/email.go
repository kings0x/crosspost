package email

import (
	"fmt"
	"log"
	"net/smtp"

	"github.com/kings0x/crossPost/internal/config"
)

// SendVerificationEmail sends a verification email with a magic link.
// If SMTP settings are not configured, it logs the verification link for development.
func SendVerificationEmail(cfg *config.Config, to, link string) error {
	host := cfg.SMTP_HOST
	port := cfg.SMTP_PORT
	user := cfg.SMTP_USER
	pass := cfg.SMTP_PASS
	from := cfg.FROM_EMAIL

	if host == "" || port == "" || user == "" || pass == "" || from == "" {
		log.Printf("verification link for %s: %s", to, link)
		return nil
	}

	addr := fmt.Sprintf("%s:%s", host, port)
	auth := smtp.PlainAuth("", user, pass, host)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: Verify your email\r\n\r\nPlease click the link to verify: %s\r\n", from, to, link)

	if err := smtp.SendMail(addr, auth, from, []string{to}, []byte(msg)); err != nil {
		return fmt.Errorf("SendVerificationEmail: %w", err)
	}

	return nil
}
