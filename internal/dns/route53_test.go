package dns

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/route53"
	"gotest.tools/assert"
)

type mockRoute53Client struct {
	changeResourceRecordSetsFunc func(ctx context.Context, params *route53.ChangeResourceRecordSetsInput, optFns ...func(*route53.Options)) (*route53.ChangeResourceRecordSetsOutput, error)
}

func (m *mockRoute53Client) ChangeResourceRecordSets(ctx context.Context, params *route53.ChangeResourceRecordSetsInput, optFns ...func(*route53.Options)) (*route53.ChangeResourceRecordSetsOutput, error) {
	if m.changeResourceRecordSetsFunc != nil {
		return m.changeResourceRecordSetsFunc(ctx, params, optFns...)
	}
	return &route53.ChangeResourceRecordSetsOutput{}, nil
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestNewRoute53Service(t *testing.T) {
	client := &mockRoute53Client{}
	logger := newTestLogger()

	svc := NewRoute53Service(client, "Z123456789", logger)

	assert.Assert(t, svc != nil)
	assert.Equal(t, "Z123456789", svc.hostedZoneID)
}

func TestRoute53Service_UpsertRecord_Success(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	var capturedInput *route53.ChangeResourceRecordSetsInput
	client := &mockRoute53Client{
		changeResourceRecordSetsFunc: func(ctx context.Context, params *route53.ChangeResourceRecordSetsInput, optFns ...func(*route53.Options)) (*route53.ChangeResourceRecordSetsOutput, error) {
			capturedInput = params
			return &route53.ChangeResourceRecordSetsOutput{}, nil
		},
	}

	svc := NewRoute53Service(client, "Z123456789", logger)

	err := svc.UpsertRecord(ctx, "a3f8c2d1", "203.0.113.42")

	assert.NilError(t, err)
	assert.Assert(t, capturedInput != nil)
	assert.Equal(t, "Z123456789", *capturedInput.HostedZoneId)
	assert.Assert(t, len(capturedInput.ChangeBatch.Changes) == 1)

	change := capturedInput.ChangeBatch.Changes[0]
	assert.Equal(t, "a3f8c2d1."+RootDomain, *change.ResourceRecordSet.Name)
	assert.Equal(t, int64(DefaultTTL), *change.ResourceRecordSet.TTL)
	assert.Assert(t, len(change.ResourceRecordSet.ResourceRecords) == 1)
	assert.Equal(t, "203.0.113.42", *change.ResourceRecordSet.ResourceRecords[0].Value)
}

func TestRoute53Service_UpsertRecord_Error(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	client := &mockRoute53Client{
		changeResourceRecordSetsFunc: func(ctx context.Context, params *route53.ChangeResourceRecordSetsInput, optFns ...func(*route53.Options)) (*route53.ChangeResourceRecordSetsOutput, error) {
			return nil, errors.New("route53 error")
		},
	}

	svc := NewRoute53Service(client, "Z123456789", logger)

	err := svc.UpsertRecord(ctx, "a3f8c2d1", "203.0.113.42")

	assert.Assert(t, err != nil)
	assert.ErrorContains(t, err, "failed to upsert DNS record")
}

func TestRoute53Service_DeleteRecord(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()
	client := &mockRoute53Client{}

	svc := NewRoute53Service(client, "Z123456789", logger)

	// DeleteRecord is not fully implemented, should return nil
	err := svc.DeleteRecord(ctx, "a3f8c2d1")

	assert.NilError(t, err)
}

func TestDefaultTTL(t *testing.T) {
	assert.Equal(t, int64(300), int64(DefaultTTL))
}

func TestRoute53Service_ImplementsService(t *testing.T) {
	var _ Service = (*Route53Service)(nil)
}
