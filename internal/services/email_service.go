package services

import (
	"fmt"
	"log"

	"github.com/rafabene/avantpro-backend/internal/models"
)

// EmailServiceInterface defines the interface for email operations.
// This interface provides methods for sending various types of emails
// related to organization management and user notifications.
type EmailServiceInterface interface {
	// SendOrganizationInvite sends an invitation email to a user to join an organization.
	// The email contains a unique invitation link that allows the recipient to accept
	// the invitation and join the organization with the specified role.
	// Parameters:
	//   - invite: The organization invitation containing recipient email, organization details, and token
	//   - baseURL: The base URL of the application for generating the invitation acceptance link
	// Returns:
	//   - error: Error if email sending fails
	SendOrganizationInvite(invite *models.OrganizationInvite, baseURL string) error

	// SendPasswordResetEmail sends a password reset email to a user.
	// The email contains a unique reset link that allows the user to reset their password.
	// Parameters:
	//   - email: The email address of the user requesting password reset
	//   - resetToken: The unique token for password reset
	//   - baseURL: The base URL of the application for generating the reset link
	// Returns:
	//   - error: Error if email sending fails
	SendPasswordResetEmail(email, resetToken, baseURL string) error
}

// EmailService implements the email service interface.
// This is currently a basic implementation that logs email content instead of
// actually sending emails. In production, this should be replaced with a proper
// email service provider like SendGrid, AWS SES, or SMTP configuration.
//
// Future enhancements should include:
//   - SMTP server configuration
//   - HTML email templates
//   - Email delivery tracking
//   - Retry mechanisms for failed deliveries
//   - Email queue processing for high volume
type EmailService struct {
	// For now, this is a simple logging implementation
	// In production, you would add fields like:
	// smtpHost     string
	// smtpPort     int
	// smtpUsername string
	// smtpPassword string
	// templates    map[string]*template.Template
}

// NewEmailService creates a new instance of EmailService.
// This constructor initializes the email service with default configuration.
// In production, this would accept configuration parameters for SMTP settings,
// template paths, and other email provider settings.
// Returns:
//   - EmailServiceInterface: Configured email service ready for use
func NewEmailService() EmailServiceInterface {
	return &EmailService{}
}

// SendOrganizationInvite sends an invitation email to join an organization.
// This method creates and sends an email invitation containing:
//   - Organization name and role information
//   - Unique invitation link for acceptance
//   - Expiration date and time
//   - Instructions for accepting or ignoring the invitation
//
// Currently this method logs the email content instead of sending actual emails.
// In production, this should be replaced with actual email delivery logic.
//
// Parameters:
//   - invite: Organization invitation model containing all necessary data
//   - baseURL: Base application URL for generating the acceptance link
//
// Returns:
//   - error: Error if email composition or sending fails
func (s *EmailService) SendOrganizationInvite(invite *models.OrganizationInvite, baseURL string) error {
	// For now, we'll just log the email content
	// In production, you would use a proper email service like SendGrid, AWS SES, etc.

	inviteURL := fmt.Sprintf("%s/organizations/invites/%s/accept", baseURL, invite.Token)

	emailContent := fmt.Sprintf(`
Subject: Invitation to join %s

Dear %s,

You have been invited to join the organization "%s" as a %s.

To accept this invitation, please click on the following link:
%s

This invitation will expire on %s.

If you didn't expect this invitation, you can safely ignore this email.

Best regards,
The AvantPro Team
	`,
		invite.Organization.Name,
		invite.Email,
		invite.Organization.Name,
		invite.Role,
		inviteURL,
		invite.ExpiresAt.Format("2006-01-02 15:04:05"),
	)

	// TODO: In production, replace this with actual email sending
	log.Printf("\n\n====== ORGANIZATION INVITE EMAIL ======")
	log.Printf("To: %s", invite.Email)
	log.Printf("Subject: Invitation to join %s", invite.Organization.Name)
	log.Printf("Content:\n%s", emailContent)
	log.Printf("========================================\n")

	return nil
}

// SendPasswordResetEmail sends a password reset email to a user.
// This method creates and sends a password reset email containing:
//   - Instructions for resetting the password
//   - Unique reset link with token
//   - Security notice about password reset request
//
// Currently this method logs the email content instead of sending actual emails.
// In production, this should be replaced with actual email delivery logic.
//
// Parameters:
//   - email: Email address of the user requesting password reset
//   - resetToken: Unique token for password reset verification
//   - baseURL: Base application URL for generating the reset link
//
// Returns:
//   - error: Error if email composition or sending fails
func (s *EmailService) SendPasswordResetEmail(email, resetToken, baseURL string) error {
	// Generate password reset URL
	resetURL := fmt.Sprintf("%s/auth/password-reset/confirm?token=%s", baseURL, resetToken)

	emailContent := fmt.Sprintf(`
Subject: Password Reset Request

Dear user,

You have requested to reset your password for your AvantPro account.

To reset your password, please click on the following link:
%s

This link will expire in 1 hour for security reasons.

If you didn't request this password reset, please ignore this email. Your password will remain unchanged.

For security reasons, please do not share this link with anyone.

Best regards,
The AvantPro Team
	`,
		resetURL,
	)

	// TODO: In production, replace this with actual email sending
	log.Printf("\n\n====== PASSWORD RESET EMAIL ======")
	log.Printf("To: %s", email)
	log.Printf("Subject: Password Reset Request")
	log.Printf("Content:\n%s", emailContent)
	log.Printf("Reset Token: %s", resetToken)
	log.Printf("===================================\n")

	return nil
}
