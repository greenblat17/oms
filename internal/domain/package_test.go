package domain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateWeight(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		packageType OrderPackageType
		weight      float64
		expectedErr error
	}{
		{
			name:        "Default package, valid weight",
			packageType: OrderPackageType{OrderPackager: newDefaultPackage()},
			weight:      5,
			expectedErr: nil,
		},
		{
			name:        "Default package, negative weight",
			packageType: OrderPackageType{OrderPackager: newDefaultPackage()},
			weight:      -5,
			expectedErr: ErrWeightNegative,
		},
		{
			name:        "Standard package, valid weight",
			packageType: OrderPackageType{OrderPackager: newStandardPackage()},
			weight:      5,
			expectedErr: nil,
		},
		{
			name:        "Standard package, negative weight",
			packageType: OrderPackageType{OrderPackager: newStandardPackage()},
			weight:      -5,
			expectedErr: ErrWeightNegative,
		},
		{
			name:        "Standard package, exceeding weight",
			packageType: OrderPackageType{OrderPackager: newStandardPackage()},
			weight:      15,
			expectedErr: ErrWeightExceedsLimit{
				Weight: 15,
				Limit:  PackageMaxWeight,
			},
		},
		{
			name:        "Box package, valid weight",
			packageType: OrderPackageType{OrderPackager: newBoxPackage()},
			weight:      20,
			expectedErr: nil,
		},
		{
			name:        "Box package, exceeding weight",
			packageType: OrderPackageType{OrderPackager: newBoxPackage()},
			weight:      35,
			expectedErr: ErrWeightExceedsLimit{
				Weight: 35,
				Limit:  BoxMaxWeight,
			},
		},
		{
			name:        "Box package, negative weight",
			packageType: OrderPackageType{OrderPackager: newBoxPackage()},
			weight:      -5,
			expectedErr: ErrWeightNegative,
		},
		{
			name:        "Film package, valid weight",
			packageType: OrderPackageType{OrderPackager: newFilmPackage()},
			weight:      5,
			expectedErr: nil,
		},
		{
			name:        "Film package, negative weight",
			packageType: OrderPackageType{OrderPackager: newFilmPackage()},
			weight:      -5,
			expectedErr: ErrWeightNegative,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.packageType.ValidateWeight(tt.weight)
			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.expectedErr), "expected error: %v, got: %v", tt.expectedErr, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestNewPackageType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		packageType string
		want        *OrderPackageType
		expectError error
	}{
		{
			name:        "default package",
			packageType: "without package",
			want:        &OrderPackageType{OrderPackager: &Default{}},
			expectError: nil,
		},
		{
			name:        "standard package",
			packageType: "package",
			want:        &OrderPackageType{OrderPackager: &StandardPackage{}},
			expectError: nil,
		},
		{
			name:        "box package",
			packageType: "box",
			want:        &OrderPackageType{OrderPackager: &Box{}},
			expectError: nil,
		},
		{
			name:        "film package",
			packageType: "film",
			want:        &OrderPackageType{OrderPackager: &Film{}},
			expectError: nil,
		},
		{
			name:        "unsupported package",
			packageType: "unsupported",
			want:        nil,
			expectError: ErrPackageTypeUnsupported,
		},
		{
			name:        "empty package type",
			packageType: "",
			want:        &OrderPackageType{OrderPackager: &Default{}},
			expectError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packageType, err := NewPackageType(tt.packageType)
			if tt.expectError != nil {
				require.ErrorIs(t, err, ErrPackageTypeUnsupported, "expected error: %v, got: %v", tt.expectError, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, packageType)
			assert.Equal(t, tt.want.OrderPackager.Type(), packageType.OrderPackager.Type())
		})
	}
}
