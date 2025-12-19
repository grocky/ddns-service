package domain

import (
	"errors"
	"testing"

	"gotest.tools/assert"
)

func TestCreateOwnerRequest_Validate(t *testing.T) {
	testCases := []struct {
		name        string
		request     CreateOwnerRequest
		expectedErr error
	}{
		{
			name: "valid request",
			request: CreateOwnerRequest{
				OwnerID: "my-home-lab",
				Email:   "user@example.com",
			},
			expectedErr: nil,
		},
		{
			name: "valid with subdomain email",
			request: CreateOwnerRequest{
				OwnerID: "test-owner",
				Email:   "user@mail.example.com",
			},
			expectedErr: nil,
		},
		{
			name: "missing ownerId",
			request: CreateOwnerRequest{
				OwnerID: "",
				Email:   "user@example.com",
			},
			expectedErr: ErrMissingOwnerID,
		},
		{
			name: "whitespace-only ownerId",
			request: CreateOwnerRequest{
				OwnerID: "   ",
				Email:   "user@example.com",
			},
			expectedErr: ErrMissingOwnerID,
		},
		{
			name: "missing email",
			request: CreateOwnerRequest{
				OwnerID: "my-home-lab",
				Email:   "",
			},
			expectedErr: ErrMissingEmail,
		},
		{
			name: "whitespace-only email",
			request: CreateOwnerRequest{
				OwnerID: "my-home-lab",
				Email:   "   ",
			},
			expectedErr: ErrMissingEmail,
		},
		{
			name: "invalid email - no @",
			request: CreateOwnerRequest{
				OwnerID: "my-home-lab",
				Email:   "userexample.com",
			},
			expectedErr: ErrInvalidEmail,
		},
		{
			name: "invalid email - no domain",
			request: CreateOwnerRequest{
				OwnerID: "my-home-lab",
				Email:   "user@",
			},
			expectedErr: ErrInvalidEmail,
		},
		{
			name: "invalid email - no local part",
			request: CreateOwnerRequest{
				OwnerID: "my-home-lab",
				Email:   "@example.com",
			},
			expectedErr: ErrInvalidEmail,
		},
		{
			name: "invalid email - no TLD",
			request: CreateOwnerRequest{
				OwnerID: "my-home-lab",
				Email:   "user@example",
			},
			expectedErr: ErrInvalidEmail,
		},
		{
			name: "invalid email - dot at end",
			request: CreateOwnerRequest{
				OwnerID: "my-home-lab",
				Email:   "user@example.",
			},
			expectedErr: ErrInvalidEmail,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.request.Validate()
			if tc.expectedErr == nil {
				assert.NilError(t, err)
			} else {
				assert.Assert(t, errors.Is(err, tc.expectedErr), "expected %v, got %v", tc.expectedErr, err)
			}
		})
	}
}

func TestRecoverKeyRequest_Validate(t *testing.T) {
	testCases := []struct {
		name        string
		request     RecoverKeyRequest
		expectedErr error
	}{
		{
			name: "valid request",
			request: RecoverKeyRequest{
				Email: "user@example.com",
			},
			expectedErr: nil,
		},
		{
			name: "missing email",
			request: RecoverKeyRequest{
				Email: "",
			},
			expectedErr: ErrMissingEmail,
		},
		{
			name: "whitespace-only email",
			request: RecoverKeyRequest{
				Email: "   ",
			},
			expectedErr: ErrMissingEmail,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.request.Validate()
			if tc.expectedErr == nil {
				assert.NilError(t, err)
			} else {
				assert.Assert(t, errors.Is(err, tc.expectedErr), "expected %v, got %v", tc.expectedErr, err)
			}
		})
	}
}

func TestIsValidEmail(t *testing.T) {
	testCases := []struct {
		email    string
		expected bool
	}{
		{"user@example.com", true},
		{"user@mail.example.com", true},
		{"user.name@example.com", true},
		{"user+tag@example.com", true},
		{"user@example.co.uk", true},
		{"", false},
		{"user", false},
		{"user@", false},
		{"@example.com", false},
		{"user@example", false},
		{"user@.", false},
		{"user@example.", false},
		{"@", false},
		{"user@@example.com", true}, // Basic validation allows this
	}

	for _, tc := range testCases {
		t.Run(tc.email, func(t *testing.T) {
			result := isValidEmail(tc.email)
			assert.Equal(t, tc.expected, result)
		})
	}
}
