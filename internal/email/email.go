package email

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
)

const (
	// DefaultSenderEmail is the default email address to send from.
	// This should be verified in SES.
	DefaultSenderEmail = "noreply@rockygray.com"

	// EmailSubject is the subject line for API key emails.
	EmailSubject = "Your DDNS Service API Key"
)

// Service defines the interface for sending emails.
type Service interface {
	// SendAPIKey sends an API key to the specified email address.
	SendAPIKey(ctx context.Context, toEmail, ownerID, apiKey string) error
}

// SESClient defines the interface for SES operations we use.
type SESClient interface {
	SendEmail(ctx context.Context, params *ses.SendEmailInput, optFns ...func(*ses.Options)) (*ses.SendEmailOutput, error)
}

// SESService implements Service using AWS SES.
type SESService struct {
	client      SESClient
	senderEmail string
	logger      *slog.Logger
}

// NewSESService creates a new SES email service.
func NewSESService(client SESClient, logger *slog.Logger) *SESService {
	return &SESService{
		client:      client,
		senderEmail: DefaultSenderEmail,
		logger:      logger,
	}
}

// NewSESServiceWithSender creates a new SES email service with a custom sender.
func NewSESServiceWithSender(client SESClient, senderEmail string, logger *slog.Logger) *SESService {
	return &SESService{
		client:      client,
		senderEmail: senderEmail,
		logger:      logger,
	}
}

// SendAPIKey sends an API key to the specified email address.
func (s *SESService) SendAPIKey(ctx context.Context, toEmail, ownerID, apiKey string) error {
	body := buildAPIKeyEmailBody(ownerID, apiKey)

	input := &ses.SendEmailInput{
		Source: aws.String(s.senderEmail),
		Destination: &types.Destination{
			ToAddresses: []string{toEmail},
		},
		Message: &types.Message{
			Subject: &types.Content{
				Data:    aws.String(EmailSubject),
				Charset: aws.String("UTF-8"),
			},
			Body: &types.Body{
				Text: &types.Content{
					Data:    aws.String(body),
					Charset: aws.String("UTF-8"),
				},
				Html: &types.Content{
					Data:    aws.String(buildAPIKeyEmailHTML(ownerID, apiKey)),
					Charset: aws.String("UTF-8"),
				},
			},
		},
	}

	_, err := s.client.SendEmail(ctx, input)
	if err != nil {
		s.logger.Error("failed to send email", "error", err, "toEmail", toEmail)
		return fmt.Errorf("failed to send email: %w", err)
	}

	s.logger.Info("API key email sent", "toEmail", toEmail, "ownerId", ownerID)
	return nil
}

func buildAPIKeyEmailBody(ownerID, apiKey string) string {
	return fmt.Sprintf(`Your DDNS Service API Key

Owner ID: %s

Your new API key is:
%s

IMPORTANT: Save this key securely. It will not be shown again.

Use this key in the Authorization header for all API requests:
Authorization: Bearer %s

Example usage:
curl -X POST https://ddns.rockygray.com/register \
  -H "Authorization: Bearer %s" \
  -H "Content-Type: application/json" \
  -d '{"ownerId":"%s","location":"home","ip":"auto"}'

If you did not request this key, please ignore this email.

---
DDNS Service
https://ddns.rockygray.com
`, ownerID, apiKey, apiKey, apiKey, ownerID)
}

func buildAPIKeyEmailHTML(ownerID, apiKey string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; }
    .container { max-width: 600px; margin: 0 auto; padding: 20px; }
    .key-box { background: #f5f5f5; border: 1px solid #ddd; border-radius: 4px; padding: 15px; margin: 20px 0; font-family: monospace; word-break: break-all; }
    .warning { background: #fff3cd; border: 1px solid #ffc107; border-radius: 4px; padding: 15px; margin: 20px 0; }
    code { background: #f5f5f5; padding: 2px 6px; border-radius: 3px; font-family: monospace; }
    pre { background: #f5f5f5; padding: 15px; border-radius: 4px; overflow-x: auto; }
  </style>
</head>
<body>
  <div class="container">
    <h1>Your DDNS Service API Key</h1>

    <p><strong>Owner ID:</strong> <code>%s</code></p>

    <p>Your new API key is:</p>
    <div class="key-box">%s</div>

    <div class="warning">
      <strong>Important:</strong> Save this key securely. It will not be shown again.
    </div>

    <h2>Usage</h2>
    <p>Include this key in the Authorization header for all API requests:</p>
    <pre>Authorization: Bearer %s</pre>

    <h3>Example</h3>
    <pre>curl -X POST https://ddns.rockygray.com/register \
  -H "Authorization: Bearer %s" \
  -H "Content-Type: application/json" \
  -d '{"ownerId":"%s","location":"home","ip":"auto"}'</pre>

    <p style="color: #666; font-size: 14px; margin-top: 40px;">
      If you did not request this key, please ignore this email.
    </p>

    <hr style="border: none; border-top: 1px solid #ddd; margin: 30px 0;">
    <p style="color: #999; font-size: 12px;">
      DDNS Service<br>
      <a href="https://ddns.rockygray.com">https://ddns.rockygray.com</a>
    </p>
  </div>
</body>
</html>`, ownerID, apiKey, apiKey, apiKey, ownerID)
}

// Ensure SESService implements Service.
var _ Service = (*SESService)(nil)
