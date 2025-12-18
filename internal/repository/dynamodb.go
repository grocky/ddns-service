package repository

import (
	"context"
	"errors"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/grocky/ddns-service/internal/domain"
)

const tableName = "DdnsServiceIpMapping"

// DynamoDBClient defines the interface for DynamoDB operations we use.
type DynamoDBClient interface {
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
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
		TableName: aws.String(tableName),
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
		TableName: aws.String(tableName),
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

// Ensure DynamoDBRepository implements Repository.
var _ Repository = (*DynamoDBRepository)(nil)

// IsMappingNotFound checks if an error is a mapping not found error.
func IsMappingNotFound(err error) bool {
	return errors.Is(err, domain.ErrMappingNotFound)
}
