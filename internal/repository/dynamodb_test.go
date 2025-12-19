package repository

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/grocky/ddns-service/internal/domain"
	"gotest.tools/assert"
)

// mockDynamoDBClient is a mock implementation of DynamoDBClient for testing.
type mockDynamoDBClient struct {
	putItemFunc    func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	getItemFunc    func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	updateItemFunc func(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
}

func (m *mockDynamoDBClient) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	if m.putItemFunc != nil {
		return m.putItemFunc(ctx, params, optFns...)
	}
	return &dynamodb.PutItemOutput{}, nil
}

func (m *mockDynamoDBClient) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	if m.getItemFunc != nil {
		return m.getItemFunc(ctx, params, optFns...)
	}
	return &dynamodb.GetItemOutput{}, nil
}

func (m *mockDynamoDBClient) UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	if m.updateItemFunc != nil {
		return m.updateItemFunc(ctx, params, optFns...)
	}
	return &dynamodb.UpdateItemOutput{}, nil
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

// =============================================================================
// IP Mapping Tests
// =============================================================================

func TestDynamoDBRepository_Put(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	mapping := domain.IPMapping{
		OwnerID:      "test-owner",
		LocationName: "home",
		IP:           "192.168.1.100",
		UpdatedAt:    time.Now().UTC(),
	}

	client := &mockDynamoDBClient{
		putItemFunc: func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			assert.Equal(t, mappingsTableName, *params.TableName)
			assert.Assert(t, params.Item != nil)
			return &dynamodb.PutItemOutput{}, nil
		},
	}

	repo := NewDynamoDBRepository(client, logger)
	err := repo.Put(ctx, mapping)
	assert.NilError(t, err)
}

func TestDynamoDBRepository_Put_Error(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	mapping := domain.IPMapping{
		OwnerID:      "test-owner",
		LocationName: "home",
		IP:           "192.168.1.100",
		UpdatedAt:    time.Now().UTC(),
	}

	expectedErr := errors.New("dynamodb error")
	client := &mockDynamoDBClient{
		putItemFunc: func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return nil, expectedErr
		},
	}

	repo := NewDynamoDBRepository(client, logger)
	err := repo.Put(ctx, mapping)
	assert.Assert(t, errors.Is(err, expectedErr), "expected %v, got %v", expectedErr, err)
}

func TestDynamoDBRepository_Get(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	expectedMapping := domain.IPMapping{
		OwnerID:      "test-owner",
		LocationName: "home",
		IP:           "192.168.1.100",
		UpdatedAt:    time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	item, _ := attributevalue.MarshalMap(expectedMapping)

	client := &mockDynamoDBClient{
		getItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			assert.Equal(t, mappingsTableName, *params.TableName)
			// Verify key structure
			ownerKey := params.Key["OwnerId"].(*types.AttributeValueMemberS)
			locationKey := params.Key["LocationName"].(*types.AttributeValueMemberS)
			assert.Equal(t, "test-owner", ownerKey.Value)
			assert.Equal(t, "home", locationKey.Value)
			return &dynamodb.GetItemOutput{Item: item}, nil
		},
	}

	repo := NewDynamoDBRepository(client, logger)
	mapping, err := repo.Get(ctx, "test-owner", "home")

	assert.NilError(t, err)
	assert.Equal(t, expectedMapping.OwnerID, mapping.OwnerID)
	assert.Equal(t, expectedMapping.LocationName, mapping.LocationName)
	assert.Equal(t, expectedMapping.IP, mapping.IP)
}

func TestDynamoDBRepository_Get_NotFound(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	client := &mockDynamoDBClient{
		getItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: nil}, nil
		},
	}

	repo := NewDynamoDBRepository(client, logger)
	mapping, err := repo.Get(ctx, "nonexistent", "home")

	assert.Assert(t, mapping == nil)
	assert.Assert(t, errors.Is(err, domain.ErrMappingNotFound), "expected ErrMappingNotFound, got %v", err)
}

func TestDynamoDBRepository_Get_Error(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	expectedErr := errors.New("dynamodb error")
	client := &mockDynamoDBClient{
		getItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return nil, expectedErr
		},
	}

	repo := NewDynamoDBRepository(client, logger)
	mapping, err := repo.Get(ctx, "test-owner", "home")

	assert.Assert(t, mapping == nil)
	assert.Assert(t, errors.Is(err, expectedErr), "expected %v, got %v", expectedErr, err)
}

// =============================================================================
// Owner Tests
// =============================================================================

func TestDynamoDBRepository_CreateOwner(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	owner := domain.Owner{
		OwnerID:    "test-owner",
		Email:      "user@example.com",
		APIKeyHash: "somehash123",
		CreatedAt:  time.Now().UTC(),
	}

	client := &mockDynamoDBClient{
		putItemFunc: func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			assert.Equal(t, ownersTableName, *params.TableName)
			assert.Assert(t, params.ConditionExpression != nil)
			assert.Equal(t, "attribute_not_exists(OwnerId)", *params.ConditionExpression)
			return &dynamodb.PutItemOutput{}, nil
		},
	}

	repo := NewDynamoDBRepository(client, logger)
	err := repo.CreateOwner(ctx, owner)
	assert.NilError(t, err)
}

func TestDynamoDBRepository_CreateOwner_AlreadyExists(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	owner := domain.Owner{
		OwnerID:    "existing-owner",
		Email:      "user@example.com",
		APIKeyHash: "somehash123",
		CreatedAt:  time.Now().UTC(),
	}

	client := &mockDynamoDBClient{
		putItemFunc: func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return nil, &types.ConditionalCheckFailedException{
				Message: aws.String("The conditional request failed"),
			}
		},
	}

	repo := NewDynamoDBRepository(client, logger)
	err := repo.CreateOwner(ctx, owner)
	assert.Assert(t, errors.Is(err, domain.ErrOwnerExists), "expected ErrOwnerExists, got %v", err)
}

func TestDynamoDBRepository_CreateOwner_Error(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	owner := domain.Owner{
		OwnerID:    "test-owner",
		Email:      "user@example.com",
		APIKeyHash: "somehash123",
		CreatedAt:  time.Now().UTC(),
	}

	expectedErr := errors.New("dynamodb error")
	client := &mockDynamoDBClient{
		putItemFunc: func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return nil, expectedErr
		},
	}

	repo := NewDynamoDBRepository(client, logger)
	err := repo.CreateOwner(ctx, owner)
	assert.Assert(t, errors.Is(err, expectedErr), "expected %v, got %v", expectedErr, err)
}

func TestDynamoDBRepository_GetOwner(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	expectedOwner := domain.Owner{
		OwnerID:    "test-owner",
		Email:      "user@example.com",
		APIKeyHash: "somehash123",
		CreatedAt:  time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	item, _ := attributevalue.MarshalMap(expectedOwner)

	client := &mockDynamoDBClient{
		getItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			assert.Equal(t, ownersTableName, *params.TableName)
			ownerKey := params.Key["OwnerId"].(*types.AttributeValueMemberS)
			assert.Equal(t, "test-owner", ownerKey.Value)
			return &dynamodb.GetItemOutput{Item: item}, nil
		},
	}

	repo := NewDynamoDBRepository(client, logger)
	owner, err := repo.GetOwner(ctx, "test-owner")

	assert.NilError(t, err)
	assert.Equal(t, expectedOwner.OwnerID, owner.OwnerID)
	assert.Equal(t, expectedOwner.Email, owner.Email)
	assert.Equal(t, expectedOwner.APIKeyHash, owner.APIKeyHash)
}

func TestDynamoDBRepository_GetOwner_NotFound(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	client := &mockDynamoDBClient{
		getItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: nil}, nil
		},
	}

	repo := NewDynamoDBRepository(client, logger)
	owner, err := repo.GetOwner(ctx, "nonexistent")

	assert.Assert(t, owner == nil)
	assert.Assert(t, errors.Is(err, domain.ErrOwnerNotFound), "expected ErrOwnerNotFound, got %v", err)
}

func TestDynamoDBRepository_GetOwner_Error(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	expectedErr := errors.New("dynamodb error")
	client := &mockDynamoDBClient{
		getItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return nil, expectedErr
		},
	}

	repo := NewDynamoDBRepository(client, logger)
	owner, err := repo.GetOwner(ctx, "test-owner")

	assert.Assert(t, owner == nil)
	assert.Assert(t, errors.Is(err, expectedErr), "expected %v, got %v", expectedErr, err)
}

func TestDynamoDBRepository_UpdateOwnerKey(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	client := &mockDynamoDBClient{
		updateItemFunc: func(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
			assert.Equal(t, ownersTableName, *params.TableName)
			assert.Assert(t, params.UpdateExpression != nil)
			assert.Equal(t, "SET ApiKeyHash = :hash", *params.UpdateExpression)
			assert.Assert(t, params.ConditionExpression != nil)
			assert.Equal(t, "attribute_exists(OwnerId)", *params.ConditionExpression)

			// Verify the new hash value is passed
			hashVal := params.ExpressionAttributeValues[":hash"].(*types.AttributeValueMemberS)
			assert.Equal(t, "newhash456", hashVal.Value)

			return &dynamodb.UpdateItemOutput{}, nil
		},
	}

	repo := NewDynamoDBRepository(client, logger)
	err := repo.UpdateOwnerKey(ctx, "test-owner", "newhash456")
	assert.NilError(t, err)
}

func TestDynamoDBRepository_UpdateOwnerKey_NotFound(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	client := &mockDynamoDBClient{
		updateItemFunc: func(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
			return nil, &types.ConditionalCheckFailedException{
				Message: aws.String("The conditional request failed"),
			}
		},
	}

	repo := NewDynamoDBRepository(client, logger)
	err := repo.UpdateOwnerKey(ctx, "nonexistent", "newhash456")
	assert.Assert(t, errors.Is(err, domain.ErrOwnerNotFound), "expected ErrOwnerNotFound, got %v", err)
}

func TestDynamoDBRepository_UpdateOwnerKey_Error(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	expectedErr := errors.New("dynamodb error")
	client := &mockDynamoDBClient{
		updateItemFunc: func(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
			return nil, expectedErr
		},
	}

	repo := NewDynamoDBRepository(client, logger)
	err := repo.UpdateOwnerKey(ctx, "test-owner", "newhash456")
	assert.Assert(t, errors.Is(err, expectedErr), "expected %v, got %v", expectedErr, err)
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestIsOwnerNotFound(t *testing.T) {
	assert.Assert(t, IsOwnerNotFound(domain.ErrOwnerNotFound))
	assert.Assert(t, !IsOwnerNotFound(domain.ErrOwnerExists))
	assert.Assert(t, !IsOwnerNotFound(errors.New("some other error")))
	assert.Assert(t, !IsOwnerNotFound(nil))
}

func TestIsOwnerExists(t *testing.T) {
	assert.Assert(t, IsOwnerExists(domain.ErrOwnerExists))
	assert.Assert(t, !IsOwnerExists(domain.ErrOwnerNotFound))
	assert.Assert(t, !IsOwnerExists(errors.New("some other error")))
	assert.Assert(t, !IsOwnerExists(nil))
}

func TestIsMappingNotFound(t *testing.T) {
	assert.Assert(t, IsMappingNotFound(domain.ErrMappingNotFound))
	assert.Assert(t, !IsMappingNotFound(domain.ErrOwnerNotFound))
	assert.Assert(t, !IsMappingNotFound(errors.New("some other error")))
	assert.Assert(t, !IsMappingNotFound(nil))
}
