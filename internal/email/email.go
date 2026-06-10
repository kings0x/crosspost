package email

import (
	"fmt"
	"log"

	"github.com/kings0x/crossPost/internal/config"
	"github.com/resend/resend-go/v2"
)

func SendVerificationEmail(cfg *config.Config, to, link string) error {
	if cfg.RESEND_API_KEY == "" {
		log.Printf("verification link for %s: %s", to, link)
		return nil
	}

	client := resend.NewClient(cfg.RESEND_API_KEY)

	fromEmail := cfg.EMAIL_FROM
	if fromEmail == "" {
		fromEmail = "onboarding@resend.dev"
	}

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

	_, err := client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("SendVerificationEmail: %w", err)
	}

	return nil
}

func SendPasswordResetEmail(cfg *config.Config, to, link string) error {
	if cfg.RESEND_API_KEY == "" {
		log.Printf("password reset link for %s: %s", to, link)
		return nil
	}

	client := resend.NewClient(cfg.RESEND_API_KEY)

	fromEmail := cfg.EMAIL_FROM
	if fromEmail == "" {
		fromEmail = "onboarding@resend.dev"
	}

	toEmail := to

	// Build the email request
	params := &resend.SendEmailRequest{
		From:    fromEmail,
		To:      []string{toEmail},
		Subject: "Reset your password - CrossPost",
		Html: fmt.Sprintf(`
			<div style="font-family: Arial, sans-serif; line-height: 1.6;">
				<h2>Reset your password</h2>
				<p>We received a request to reset your password for your CrossPost account. Click the button below to choose a new password:</p>
				<p>
					<a href="%s" style="background-color: #007bff; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">
						Reset Password
					</a>
				</p>
				<p>If the button doesn't work, you can also copy and paste this link into your browser:</p>
				<p style="word-break: break-all; color: #007bff;">%s</p>
				<hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;">
				<p style="color: #888; font-size: 12px;">If you didn't request a password reset, you can safely ignore this email. Your password will remain unchanged.</p>
			</div>
		`, link, link),
	}

	_, err := client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("SendPasswordResetEmail: %w", err)
	}

	return nil
}
