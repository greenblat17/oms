package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/dto"
)

func TestNewOrder(t *testing.T) {
	t.Parallel()

	type args struct {
		packageType *OrderPackageType
		order       *dto.Order
	}

	tests := []struct {
		name        string
		args        args
		expectedErr error
	}{
		{
			name: "Valid orderDTO without package [empty input]",
			args: args{
				packageType: &OrderPackageType{OrderPackager: newDefaultPackage()},
				order: &dto.Order{
					OrderID:      1,
					RecipientID:  10,
					Weight:       5.0,
					Cost:         100.0,
					StorageUntil: time.Now().Add(24 * time.Hour),
				},
			},
			expectedErr: nil,
		},
		{
			name: "Valid orderDTO with package",
			args: args{
				packageType: &OrderPackageType{OrderPackager: newStandardPackage()},
				order: &dto.Order{
					OrderID:      1,
					RecipientID:  10,
					Weight:       5.0,
					Cost:         100.0,
					StorageUntil: time.Now().Add(24 * time.Hour),
				},
			},
			expectedErr: nil,
		},
		{
			name: "Valid orderDTO with box package",
			args: args{
				packageType: &OrderPackageType{OrderPackager: newBoxPackage()},
				order: &dto.Order{
					OrderID:      1,
					RecipientID:  10,
					Weight:       5.0,
					Cost:         100.0,
					StorageUntil: time.Now().Add(24 * time.Hour),
				},
			},
			expectedErr: nil,
		},
		{
			name: "Valid orderDTO with film package",
			args: args{
				packageType: &OrderPackageType{OrderPackager: newFilmPackage()},
				order: &dto.Order{
					OrderID:      1,
					RecipientID:  10,
					Weight:       5.0,
					Cost:         100.0,
					StorageUntil: time.Now().Add(24 * time.Hour),
				},
			},
		},
		{
			name: "Negative weight",
			args: args{
				packageType: &OrderPackageType{OrderPackager: newFilmPackage()},
				order: &dto.Order{
					OrderID:      1,
					RecipientID:  10,
					Weight:       -5.0,
					Cost:         100.0,
					StorageUntil: time.Now().Add(24 * time.Hour),
				},
			},
			expectedErr: ErrWeightNegative,
		},
		{
			name: "Heavy weight",
			args: args{
				packageType: &OrderPackageType{OrderPackager: newBoxPackage()},
				order: &dto.Order{
					OrderID:      1,
					RecipientID:  10,
					Weight:       120.0,
					Cost:         100.0,
					StorageUntil: time.Now().Add(24 * time.Hour),
				},
			},
			expectedErr: ErrWeightExceedsLimit{
				Weight: 120.0,
				Limit:  BoxMaxWeight,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			order, err := NewOrder(tt.args.order, tt.args.packageType)

			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr, "expected error: %v, got: %v", tt.expectedErr, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, order)
			assert.Equal(t, tt.args.order.OrderID, order.ID)
			assert.Equal(t, tt.args.order.RecipientID, order.RecipientID)
			assert.Equal(t, tt.args.order.Weight, order.Weight)
			assert.Equal(t, tt.args.order.Cost, order.Cost)
			assert.Equal(t, tt.args.order.StorageUntil.UTC(), order.StorageUntil)
			assert.Equal(t, tt.args.order.IssuedAt, order.IssuedAt.Time)
			assert.Equal(t, tt.args.order.ReturnAt, order.ReturnedAt.Time)
		})
	}
}
