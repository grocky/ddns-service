package domain

import (
	"errors"
	"testing"

	"gotest.tools/assert"
)

func TestUpdateRequest_Validate(t *testing.T) {
	testCases := []struct {
		name        string
		request     UpdateRequest
		expectedErr error
	}{
		{
			name: "valid request",
			request: UpdateRequest{
				OwnerID:  "my-home-lab",
				Location: "home",
			},
			expectedErr: nil,
		},
		{
			name: "valid request with different location",
			request: UpdateRequest{
				OwnerID:  "my-home-lab",
				Location: "office",
			},
			expectedErr: nil,
		},
		{
			name: "missing ownerId",
			request: UpdateRequest{
				OwnerID:  "",
				Location: "home",
			},
			expectedErr: ErrMissingOwnerID,
		},
		{
			name: "missing location",
			request: UpdateRequest{
				OwnerID:  "my-home-lab",
				Location: "",
			},
			expectedErr: ErrMissingLocation,
		},
		{
			name: "missing both",
			request: UpdateRequest{
				OwnerID:  "",
				Location: "",
			},
			expectedErr: ErrMissingOwnerID, // OwnerID is checked first
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
