package repository

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/grocky/ddns-service/internal/domain"
)

const (
	mappingsTableName       = "DdnsServiceIpMapping"
	ownersTableName         = "DdnsServiceOwners"
	acmeChallengesTableName = "DdnsServiceAcmeChallenges"
)

// DynamoDBClient defines the interface for DynamoDB operations we use.
type DynamoDBClient interface {
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
	DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	Scan(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error)
}

// DynamoDBRepository implements Repository using DynamoDB.
type DynamoDBRepository struct {
	client DynamoDBClient
	logger *slog.Logger
}

// NewDynamoDBRepository creates a new DynamoDB repository.
func NewDynamoDBRepository(client DynamoDBClient, logger *slog.Logger) *DynamoDBRepository {
	return &DynamoDBRepository{
		client: client,
		logger: logger,
	}
}

// Put creates or updates an IP mapping in DynamoDB.
func (r *DynamoDBRepository) Put(ctx context.Context, mapping domain.IPMapping) error {
	item, err := attributevalue.MarshalMap(mapping)
	if err != nil {
		r.logger.Error("failed to marshal mapping", "error", err)
		return err
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(mappingsTableName),
		Item:      item,
	}

	_, err = r.client.PutItem(ctx, input)
	if err != nil {
		r.logger.Error("failed to put item",
			"error", err,
			"ownerId", mapping.OwnerID,
			"location", mapping.LocationName,
		)
		return err
	}

	r.logger.Info("mapping saved",
		"ownerId", mapping.OwnerID,
		"location", mapping.LocationName,
		"ip", mapping.IP,
	)
	return nil
}

// Get retrieves an IP mapping from DynamoDB.
func (r *DynamoDBRepository) Get(ctx context.Context, ownerID, location string) (*domain.IPMapping, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(mappingsTableName),
		Key: map[string]types.AttributeValue{
			"OwnerId":      &types.AttributeValueMemberS{Value: ownerID},
			"LocationName": &types.AttributeValueMemberS{Value: location},
		},
	}

	result, err := r.client.GetItem(ctx, input)
	if err != nil {
		r.logger.Error("failed to get item",
			"error", err,
			"ownerId", ownerID,
			"location", location,
		)
		return nil, err
	}

	if result.Item == nil {
		return nil, domain.ErrMappingNotFound
	}

	var mapping domain.IPMapping
	if err := attributevalue.UnmarshalMap(result.Item, &mapping); err != nil {
		r.logger.Error("failed to unmarshal mapping", "error", err)
		return nil, err
	}

	return &mapping, nil
}

// CreateOwner creates a new owner in DynamoDB.
// Uses a conditional write to fail if the owner already exists.
func (r *DynamoDBRepository) CreateOwner(ctx context.Context, owner domain.Owner) error {
	item, err := attributevalue.MarshalMap(owner)
	if err != nil {
		r.logger.Error("failed to marshal owner", "error", err)
		return err
	}

	input := &dynamodb.PutItemInput{
		TableName:           aws.String(ownersTableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(OwnerId)"),
	}

	_, err = r.client.PutItem(ctx, input)
	if err != nil {
		var condErr *types.ConditionalCheckFailedException
		if errors.As(err, &condErr) {
			return domain.ErrOwnerExists
		}
		r.logger.Error("failed to create owner", "error", err, "ownerId", owner.OwnerID)
		return err
	}

	r.logger.Info("owner created", "ownerId", owner.OwnerID)
	return nil
}

// GetOwner retrieves an owner from DynamoDB.
func (r *DynamoDBRepository) GetOwner(ctx context.Context, ownerID string) (*domain.Owner, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(ownersTableName),
		Key: map[string]types.AttributeValue{
			"OwnerId": &types.AttributeValueMemberS{Value: ownerID},
		},
	}

	result, err := r.client.GetItem(ctx, input)
	if err != nil {
		r.logger.Error("failed to get owner", "error", err, "ownerId", ownerID)
		return nil, err
	}

	if result.Item == nil {
		return nil, domain.ErrOwnerNotFound
	}

	var owner domain.Owner
	if err := attributevalue.UnmarshalMap(result.Item, &owner); err != nil {
		r.logger.Error("failed to unmarshal owner", "error", err)
		return nil, err
	}

	return &owner, nil
}

// UpdateOwnerKey updates the API key hash for an owner.
func (r *DynamoDBRepository) UpdateOwnerKey(ctx context.Context, ownerID, newKeyHash string) error {
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(ownersTableName),
		Key: map[string]types.AttributeValue{
			"OwnerId": &types.AttributeValueMemberS{Value: ownerID},
		},
		UpdateExpression: aws.String("SET ApiKeyHash = :hash"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":hash": &types.AttributeValueMemberS{Value: newKeyHash},
		},
		ConditionExpression: aws.String("attribute_exists(OwnerId)"),
	}

	_, err := r.client.UpdateItem(ctx, input)
	if err != nil {
		var condErr *types.ConditionalCheckFailedException
		if errors.As(err, &condErr) {
			return domain.ErrOwnerNotFound
		}
		r.logger.Error("failed to update owner key", "error", err, "ownerId", ownerID)
		return err
	}

	r.logger.Info("owner key updated", "ownerId", ownerID)
	return nil
}

// PutChallenge creates or updates an ACME challenge in DynamoDB.
func (r *DynamoDBRepository) PutChallenge(ctx context.Context, challenge domain.ACMEChallenge) error {
	item, err := attributevalue.MarshalMap(challenge)
	if err != nil {
		r.logger.Error("failed to marshal challenge", "error", err)
		return err
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(acmeChallengesTableName),
		Item:      item,
	}

	_, err = r.client.PutItem(ctx, input)
	if err != nil {
		r.logger.Error("failed to put challenge",
			"error", err,
			"ownerId", challenge.OwnerID,
			"location", challenge.LocationName,
		)
		return err
	}

	r.logger.Info("challenge saved",
		"ownerId", challenge.OwnerID,
		"location", challenge.LocationName,
	)
	return nil
}

// GetChallenge retrieves an ACME challenge from DynamoDB.
func (r *DynamoDBRepository) GetChallenge(ctx context.Context, ownerID, location string) (*domain.ACMEChallenge, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(acmeChallengesTableName),
		Key: map[string]types.AttributeValue{
			"OwnerId":      &types.AttributeValueMemberS{Value: ownerID},
			"LocationName": &types.AttributeValueMemberS{Value: location},
		},
	}

	result, err := r.client.GetItem(ctx, input)
	if err != nil {
		r.logger.Error("failed to get challenge",
			"error", err,
			"ownerId", ownerID,
			"location", location,
		)
		return nil, err
	}

	if result.Item == nil {
		return nil, domain.ErrChallengeNotFound
	}

	var challenge domain.ACMEChallenge
	if err := attributevalue.UnmarshalMap(result.Item, &challenge); err != nil {
		r.logger.Error("failed to unmarshal challenge", "error", err)
		return nil, err
	}

	return &challenge, nil
}

// DeleteChallenge removes an ACME challenge from DynamoDB.
func (r *DynamoDBRepository) DeleteChallenge(ctx context.Context, ownerID, location string) error {
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(acmeChallengesTableName),
		Key: map[string]types.AttributeValue{
			"OwnerId":      &types.AttributeValueMemberS{Value: ownerID},
			"LocationName": &types.AttributeValueMemberS{Value: location},
		},
	}

	_, err := r.client.DeleteItem(ctx, input)
	if err != nil {
		r.logger.Error("failed to delete challenge",
			"error", err,
			"ownerId", ownerID,
			"location", location,
		)
		return err
	}

	r.logger.Info("challenge deleted",
		"ownerId", ownerID,
		"location", location,
	)
	return nil
}

// ScanExpiredChallenges returns all ACME challenges that have expired.
// Uses a filter expression to find challenges where TTL is less than current time.
func (r *DynamoDBRepository) ScanExpiredChallenges(ctx context.Context) ([]domain.ACMEChallenge, error) {
	now := time.Now().Unix()

	input := &dynamodb.ScanInput{
		TableName:        aws.String(acmeChallengesTableName),
		FilterExpression: aws.String("#ttl < :now"),
		ExpressionAttributeNames: map[string]string{
			"#ttl": "TTL",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":now": &types.AttributeValueMemberN{Value: strconv.FormatInt(now, 10)},
		},
	}

	result, err := r.client.Scan(ctx, input)
	if err != nil {
		r.logger.Error("failed to scan expired challenges", "error", err)
		return nil, err
	}

	var challenges []domain.ACMEChallenge
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &challenges); err != nil {
		r.logger.Error("failed to unmarshal challenges", "error", err)
		return nil, err
	}

	r.logger.Info("scanned expired challenges", "count", len(challenges))
	return challenges, nil
}

// Ensure DynamoDBRepository implements Repository.
var _ Repository = (*DynamoDBRepository)(nil)
