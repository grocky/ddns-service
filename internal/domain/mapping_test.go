package domain

import (
	"errors"
	"testing"

	"gotest.tools/assert"
)

func TestRegisterRequest_Validate(t *testing.T) {
	testCases := []struct {
		name        string
		request     RegisterRequest
		expectedErr error
	}{
		{
			name: "valid request with auto IP",
			request: RegisterRequest{
				OwnerID:  "my-home-lab",
				Location: "home",
				IP:       "auto",
			},
			expectedErr: nil,
		},
		{
			name: "valid request with explicit IP",
			request: RegisterRequest{
				OwnerID:  "my-home-lab",
				Location: "office",
				IP:       "192.168.1.100",
			},
			expectedErr: nil,
		},
		{
			name: "valid request with empty IP (defaults to auto)",
			request: RegisterRequest{
				OwnerID:  "my-home-lab",
				Location: "office",
				IP:       "",
			},
			expectedErr: nil,
		},
		{
			name: "missing ownerId",
			request: RegisterRequest{
				OwnerID:  "",
				Location: "home",
				IP:       "auto",
			},
			expectedErr: ErrMissingOwnerID,
		},
		{
			name: "missing location",
			request: RegisterRequest{
				OwnerID:  "my-home-lab",
				Location: "",
				IP:       "auto",
			},
			expectedErr: ErrMissingLocation,
		},
		{
			name: "missing both",
			request: RegisterRequest{
				OwnerID:  "",
				Location: "",
				IP:       "auto",
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
