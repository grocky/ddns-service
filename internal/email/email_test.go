package email

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ses"
	"gotest.tools/assert"
)

// mockSESClient is a mock implementation of SESClient for testing.
type mockSESClient struct {
	sendEmailFunc func(ctx context.Context, params *ses.SendEmailInput, optFns ...func(*ses.Options)) (*ses.SendEmailOutput, error)
}

func (m *mockSESClient) SendEmail(ctx context.Context, params *ses.SendEmailInput, optFns ...func(*ses.Options)) (*ses.SendEmailOutput, error) {
	if m.sendEmailFunc != nil {
		return m.sendEmailFunc(ctx, params, optFns...)
	}
	return &ses.SendEmailOutput{}, nil
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestNewSESService(t *testing.T) {
	client := &mockSESClient{}
	logger := newTestLogger()

	svc := NewSESService(client, logger)

	assert.Assert(t, svc != nil)
	assert.Equal(t, DefaultSenderEmail, svc.senderEmail)
}

func TestNewSESServiceWithSender(t *testing.T) {
	client := &mockSESClient{}
	logger := newTestLogger()
	customSender := "custom@example.com"

	svc := NewSESServiceWithSender(client, customSender, logger)

	assert.Assert(t, svc != nil)
	assert.Equal(t, customSender, svc.senderEmail)
}

func TestSendAPIKey_Success(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	var capturedInput *ses.SendEmailInput

	client := &mockSESClient{
		sendEmailFunc: func(ctx context.Context, params *ses.SendEmailInput, optFns ...func(*ses.Options)) (*ses.SendEmailOutput, error) {
			capturedInput = params
			return &ses.SendEmailOutput{}, nil
		},
	}

	svc := NewSESService(client, logger)

	err := svc.SendAPIKey(ctx, "user@example.com", "test-owner", "ddns_sk_testkey123")

	assert.NilError(t, err)

	// Verify the email was constructed correctly
	assert.Equal(t, DefaultSenderEmail, *capturedInput.Source)
	assert.Equal(t, 1, len(capturedInput.Destination.ToAddresses))
	assert.Equal(t, "user@example.com", capturedInput.Destination.ToAddresses[0])
	assert.Equal(t, EmailSubject, *capturedInput.Message.Subject.Data)

	// Verify both text and HTML bodies contain the key
	assert.Assert(t, strings.Contains(*capturedInput.Message.Body.Text.Data, "ddns_sk_testkey123"))
	assert.Assert(t, strings.Contains(*capturedInput.Message.Body.Html.Data, "ddns_sk_testkey123"))

	// Verify owner ID is included
	assert.Assert(t, strings.Contains(*capturedInput.Message.Body.Text.Data, "test-owner"))
	assert.Assert(t, strings.Contains(*capturedInput.Message.Body.Html.Data, "test-owner"))
}

func TestSendAPIKey_Error(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	expectedErr := errors.New("SES error")
	client := &mockSESClient{
		sendEmailFunc: func(ctx context.Context, params *ses.SendEmailInput, optFns ...func(*ses.Options)) (*ses.SendEmailOutput, error) {
			return nil, expectedErr
		},
	}

	svc := NewSESService(client, logger)

	err := svc.SendAPIKey(ctx, "user@example.com", "test-owner", "ddns_sk_testkey123")

	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "failed to send email"))
}

func TestSendAPIKey_CustomSender(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	var capturedSource string

	client := &mockSESClient{
		sendEmailFunc: func(ctx context.Context, params *ses.SendEmailInput, optFns ...func(*ses.Options)) (*ses.SendEmailOutput, error) {
			capturedSource = *params.Source
			return &ses.SendEmailOutput{}, nil
		},
	}

	customSender := "support@mycompany.com"
	svc := NewSESServiceWithSender(client, customSender, logger)

	err := svc.SendAPIKey(ctx, "user@example.com", "owner", "key")

	assert.NilError(t, err)
	assert.Equal(t, customSender, capturedSource)
}

func TestBuildAPIKeyEmailBody(t *testing.T) {
	body := buildAPIKeyEmailBody("test-owner", "ddns_sk_abc123")

	// Verify essential content is present
	assert.Assert(t, strings.Contains(body, "test-owner"))
	assert.Assert(t, strings.Contains(body, "ddns_sk_abc123"))
	assert.Assert(t, strings.Contains(body, "Authorization: Bearer"))
	assert.Assert(t, strings.Contains(body, "IMPORTANT"))
	assert.Assert(t, strings.Contains(body, APIEndpoint))
}

func TestBuildAPIKeyEmailHTML(t *testing.T) {
	html := buildAPIKeyEmailHTML("test-owner", "ddns_sk_abc123")

	// Verify essential content is present
	assert.Assert(t, strings.Contains(html, "test-owner"))
	assert.Assert(t, strings.Contains(html, "ddns_sk_abc123"))
	assert.Assert(t, strings.Contains(html, "Authorization: Bearer"))
	assert.Assert(t, strings.Contains(html, "<!DOCTYPE html>"))
	assert.Assert(t, strings.Contains(html, "</html>"))
	assert.Assert(t, strings.Contains(html, APIEndpoint))
}

func TestSESServiceImplementsInterface(t *testing.T) {
	var _ Service = (*SESService)(nil)
}
