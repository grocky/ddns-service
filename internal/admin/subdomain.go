package admin

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	route53types "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/grocky/ddns-service/internal/dns"
)

const (
	// DefaultTTL for DNS records
	DefaultTTL = 300
)

// DynamoDBClient defines the DynamoDB operations needed.
type DynamoDBClient interface {
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
}

// Route53Client defines the Route53 operations needed.
type Route53Client interface {
	ChangeResourceRecordSets(ctx context.Context, params *route53.ChangeResourceRecordSetsInput, optFns ...func(*route53.Options)) (*route53.ChangeResourceRecordSetsOutput, error)
}

// SubdomainService handles subdomain management operations.
type SubdomainService struct {
	dynamoClient DynamoDBClient
	route53Client Route53Client
	tableName    string
	hostedZoneID string
	logger       *slog.Logger
}

// NewSubdomainService creates a new subdomain management service.
func NewSubdomainService(
	dynamoClient DynamoDBClient,
	route53Client Route53Client,
	tableName string,
	hostedZoneID string,
	logger *slog.Logger,
) *SubdomainService {
	return &SubdomainService{
		dynamoClient:  dynamoClient,
		route53Client: route53Client,
		tableName:     tableName,
		hostedZoneID:  hostedZoneID,
		logger:        logger,
	}
}

// ChangeSubdomainInput contains the parameters for changing a subdomain.
type ChangeSubdomainInput struct {
	OwnerID      string
	Location     string
	NewSubdomain string
}

// ChangeSubdomainOutput contains the result of changing a subdomain.
type ChangeSubdomainOutput struct {
	OldSubdomain string
	NewSubdomain string
	OldFQDN      string
	NewFQDN      string
	IP           string
}

// ChangeSubdomain changes the subdomain for an owner's location.
// It updates both Route53 and DynamoDB atomically.
func (s *SubdomainService) ChangeSubdomain(ctx context.Context, input ChangeSubdomainInput) (*ChangeSubdomainOutput, error) {
	s.logger.Info("changing subdomain",
		"ownerId", input.OwnerID,
		"location", input.Location,
		"newSubdomain", input.NewSubdomain,
	)

	// Step 1: Get current mapping from DynamoDB
	mapping, err := s.getMapping(ctx, input.OwnerID, input.Location)
	if err != nil {
		return nil, fmt.Errorf("failed to get mapping: %w", err)
	}

	oldSubdomain := mapping.Subdomain
	ip := mapping.IP

	if oldSubdomain == "" {
		// Use the hash-based subdomain if not set
		oldSubdomain = dns.GenerateSubdomain(input.OwnerID, input.Location)
	}

	oldFQDN := fmt.Sprintf("%s.%s", oldSubdomain, dns.RootDomain)
	newFQDN := fmt.Sprintf("%s.%s", input.NewSubdomain, dns.RootDomain)

	s.logger.Info("current state",
		"oldSubdomain", oldSubdomain,
		"newSubdomain", input.NewSubdomain,
		"ip", ip,
	)

	// Step 2: Update Route53 - delete old record, create new record
	err = s.updateRoute53(ctx, oldSubdomain, input.NewSubdomain, ip)
	if err != nil {
		return nil, fmt.Errorf("failed to update Route53: %w", err)
	}

	// Step 3: Update DynamoDB with new subdomain
	err = s.updateDynamoDB(ctx, input.OwnerID, input.Location, input.NewSubdomain)
	if err != nil {
		// Attempt to rollback Route53 changes
		s.logger.Error("DynamoDB update failed, attempting Route53 rollback", "error", err)
		rollbackErr := s.updateRoute53(ctx, input.NewSubdomain, oldSubdomain, ip)
		if rollbackErr != nil {
			s.logger.Error("Route53 rollback failed", "error", rollbackErr)
		}
		return nil, fmt.Errorf("failed to update DynamoDB: %w", err)
	}

	s.logger.Info("subdomain changed successfully",
		"oldFQDN", oldFQDN,
		"newFQDN", newFQDN,
	)

	return &ChangeSubdomainOutput{
		OldSubdomain: oldSubdomain,
		NewSubdomain: input.NewSubdomain,
		OldFQDN:      oldFQDN,
		NewFQDN:      newFQDN,
		IP:           ip,
	}, nil
}

// mappingRecord represents the relevant fields from DynamoDB.
type mappingRecord struct {
	Subdomain string
	IP        string
}

func (s *SubdomainService) getMapping(ctx context.Context, ownerID, location string) (*mappingRecord, error) {
	result, err := s.dynamoClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"OwnerId":      &types.AttributeValueMemberS{Value: ownerID},
			"LocationName": &types.AttributeValueMemberS{Value: location},
		},
	})
	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		return nil, fmt.Errorf("mapping not found for owner=%s location=%s", ownerID, location)
	}

	record := &mappingRecord{}

	if v, ok := result.Item["Subdomain"].(*types.AttributeValueMemberS); ok {
		record.Subdomain = v.Value
	}
	if v, ok := result.Item["IP"].(*types.AttributeValueMemberS); ok {
		record.IP = v.Value
	}

	if record.IP == "" {
		return nil, fmt.Errorf("mapping has no IP address")
	}

	return record, nil
}

func (s *SubdomainService) updateRoute53(ctx context.Context, oldSubdomain, newSubdomain, ip string) error {
	oldRecordName := fmt.Sprintf("%s.%s", oldSubdomain, dns.RootDomain)
	newRecordName := fmt.Sprintf("%s.%s", newSubdomain, dns.RootDomain)

	changes := []route53types.Change{
		{
			Action: route53types.ChangeActionDelete,
			ResourceRecordSet: &route53types.ResourceRecordSet{
				Name: aws.String(oldRecordName),
				Type: route53types.RRTypeA,
				TTL:  aws.Int64(DefaultTTL),
				ResourceRecords: []route53types.ResourceRecord{
					{Value: aws.String(ip)},
				},
			},
		},
		{
			Action: route53types.ChangeActionCreate,
			ResourceRecordSet: &route53types.ResourceRecordSet{
				Name: aws.String(newRecordName),
				Type: route53types.RRTypeA,
				TTL:  aws.Int64(DefaultTTL),
				ResourceRecords: []route53types.ResourceRecord{
					{Value: aws.String(ip)},
				},
			},
		},
	}

	_, err := s.route53Client.ChangeResourceRecordSets(ctx, &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(s.hostedZoneID),
		ChangeBatch: &route53types.ChangeBatch{
			Comment: aws.String(fmt.Sprintf("Change subdomain from %s to %s", oldSubdomain, newSubdomain)),
			Changes: changes,
		},
	})

	return err
}

func (s *SubdomainService) updateDynamoDB(ctx context.Context, ownerID, location, newSubdomain string) error {
	_, err := s.dynamoClient.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"OwnerId":      &types.AttributeValueMemberS{Value: ownerID},
			"LocationName": &types.AttributeValueMemberS{Value: location},
		},
		UpdateExpression: aws.String("SET Subdomain = :subdomain"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":subdomain": &types.AttributeValueMemberS{Value: newSubdomain},
		},
	})

	return err
}
