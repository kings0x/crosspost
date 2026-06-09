package email

import (
	"fmt"
	"log"

	"github.com/kings0x/crossPost/internal/config"
	"github.com/resend/resend-go/v2"
)

// SendVerificationEmail sends a verification email with a magic link using Resend.
func SendVerificationEmail(cfg *config.Config, to, link string) error {
	// If the API key is missing, fallback to logging the link (useful for local dev)
	if cfg.RESEND_API_KEY == "" {
		log.Printf("verification link for %s: %s", to, link)
		return nil
	}

	// Initialize Resend client
	client := resend.NewClient(cfg.RESEND_API_KEY)

	// Use configured FROM email, or fallback to Resend's default testing domain
	fromEmail := cfg.EMAIL_FROM
	if fromEmail == "" {
		fromEmail = "onboarding@resend.dev"
	}

	// Use the provided 'to' address. If empty, fallback to EMAIL_TO env var (great for testing)
	toEmail := to

	// Build the email request
	params := &resend.SendEmailRequest{
		From:    fromEmail,
		To:      []string{toEmail},
		Subject: "Verify your email - CrossPost",
		Html: fmt.Sprintf(`
			<div style="font-family: Arial, sans-serif; line-height: 1.6;">
				<h2>Welcome to CrossPost!</h2>
				<p>Please click the button below to verify your email address:</p>
				<p>
					<a href="%s" style="background-color: #007bff; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">
						Verify My Email
					</a>
				</p>
				<p>If the button doesn't work, you can also copy and paste this link into your browser:</p>
				<p style="word-break: break-all; color: #007bff;">%s</p>
				<hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;">
				<p style="color: #888; font-size: 12px;">If you didn't create an account, you can safely ignore this email.</p>
			</div>
		`, link, link),
	}

	// Send the email

	_, err := client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("SendVerificationEmail: %w", err)
	}

	return nil
}
