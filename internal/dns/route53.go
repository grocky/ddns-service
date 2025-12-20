package dns

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
)

const (
	// DefaultTTL is the TTL for DNS records in seconds.
	DefaultTTL = 300
)

// Service defines the interface for DNS operations.
type Service interface {
	// UpsertRecord creates or updates an A record for the given subdomain.
	UpsertRecord(ctx context.Context, subdomain, ip string) error

	// DeleteRecord removes the A record for the given subdomain.
	DeleteRecord(ctx context.Context, subdomain string) error

	// UpsertTXTRecord creates or updates a TXT record.
	UpsertTXTRecord(ctx context.Context, name, value string) error

	// DeleteTXTRecord removes a TXT record.
	DeleteTXTRecord(ctx context.Context, name, value string) error
}

// Route53Client defines the interface for Route53 operations we use.
type Route53Client interface {
	ChangeResourceRecordSets(ctx context.Context, params *route53.ChangeResourceRecordSetsInput, optFns ...func(*route53.Options)) (*route53.ChangeResourceRecordSetsOutput, error)
}

// Route53Service implements Service using AWS Route53.
type Route53Service struct {
	client       Route53Client
	hostedZoneID string
	logger       *slog.Logger
}

// NewRoute53Service creates a new Route53 DNS service.
func NewRoute53Service(client Route53Client, hostedZoneID string, logger *slog.Logger) *Route53Service {
	return &Route53Service{
		client:       client,
		hostedZoneID: hostedZoneID,
		logger:       logger,
	}
}

// UpsertRecord creates or updates an A record for the given subdomain.
func (s *Route53Service) UpsertRecord(ctx context.Context, subdomain, ip string) error {
	recordName := fmt.Sprintf("%s.%s", subdomain, RootDomain)

	input := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(s.hostedZoneID),
		ChangeBatch: &types.ChangeBatch{
			Comment: aws.String(fmt.Sprintf("DDNS update for %s", subdomain)),
			Changes: []types.Change{
				{
					Action: types.ChangeActionUpsert,
					ResourceRecordSet: &types.ResourceRecordSet{
						Name: aws.String(recordName),
						Type: types.RRTypeA,
						TTL:  aws.Int64(DefaultTTL),
						ResourceRecords: []types.ResourceRecord{
							{
								Value: aws.String(ip),
							},
						},
					},
				},
			},
		},
	}

	_, err := s.client.ChangeResourceRecordSets(ctx, input)
	if err != nil {
		s.logger.Error("failed to upsert DNS record",
			"error", err,
			"subdomain", subdomain,
			"ip", ip,
		)
		return fmt.Errorf("failed to upsert DNS record: %w", err)
	}

	s.logger.Info("DNS record upserted",
		"subdomain", subdomain,
		"recordName", recordName,
		"ip", ip,
	)
	return nil
}

// DeleteRecord removes the A record for the given subdomain.
func (s *Route53Service) DeleteRecord(ctx context.Context, subdomain string) error {
	recordName := fmt.Sprintf("%s.%s", subdomain, RootDomain)

	// To delete, we need to know the current value. For now, we'll skip this
	// as we don't have a use case for deletion yet.
	s.logger.Warn("delete record not fully implemented",
		"subdomain", subdomain,
		"recordName", recordName,
	)
	return nil
}

// UpsertTXTRecord creates or updates a TXT record.
func (s *Route53Service) UpsertTXTRecord(ctx context.Context, name, value string) error {
	recordName := fmt.Sprintf("%s.%s", name, RootDomain)

	// TXT values must be quoted in Route53
	quotedValue := fmt.Sprintf("\"%s\"", value)

	input := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(s.hostedZoneID),
		ChangeBatch: &types.ChangeBatch{
			Comment: aws.String(fmt.Sprintf("ACME challenge for %s", name)),
			Changes: []types.Change{
				{
					Action: types.ChangeActionUpsert,
					ResourceRecordSet: &types.ResourceRecordSet{
						Name: aws.String(recordName),
						Type: types.RRTypeTxt,
						TTL:  aws.Int64(DefaultTTL),
						ResourceRecords: []types.ResourceRecord{
							{Value: aws.String(quotedValue)},
						},
					},
				},
			},
		},
	}

	_, err := s.client.ChangeResourceRecordSets(ctx, input)
	if err != nil {
		s.logger.Error("failed to upsert TXT record",
			"error", err,
			"name", name,
		)
		return fmt.Errorf("failed to upsert TXT record: %w", err)
	}

	s.logger.Info("TXT record upserted",
		"name", name,
		"recordName", recordName,
	)
	return nil
}

// DeleteTXTRecord removes a TXT record.
func (s *Route53Service) DeleteTXTRecord(ctx context.Context, name, value string) error {
	recordName := fmt.Sprintf("%s.%s", name, RootDomain)
	quotedValue := fmt.Sprintf("\"%s\"", value)

	input := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(s.hostedZoneID),
		ChangeBatch: &types.ChangeBatch{
			Comment: aws.String(fmt.Sprintf("Remove ACME challenge for %s", name)),
			Changes: []types.Change{
				{
					Action: types.ChangeActionDelete,
					ResourceRecordSet: &types.ResourceRecordSet{
						Name: aws.String(recordName),
						Type: types.RRTypeTxt,
						TTL:  aws.Int64(DefaultTTL),
						ResourceRecords: []types.ResourceRecord{
							{Value: aws.String(quotedValue)},
						},
					},
				},
			},
		},
	}

	_, err := s.client.ChangeResourceRecordSets(ctx, input)
	if err != nil {
		s.logger.Error("failed to delete TXT record",
			"error", err,
			"name", name,
		)
		return fmt.Errorf("failed to delete TXT record: %w", err)
	}

	s.logger.Info("TXT record deleted",
		"name", name,
		"recordName", recordName,
	)
	return nil
}

// Ensure Route53Service implements Service.
var _ Service = (*Route53Service)(nil)
