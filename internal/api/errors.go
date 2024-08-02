package api

import (
	"errors"

	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/domain"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/module"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var errorMap = map[error]struct {
	code    codes.Code
	message string
}{
	module.ErrOrderNotFound:           {codes.NotFound, "order not found"},
	module.ErrOrderExists:             {codes.AlreadyExists, "order already exists"},
	module.ErrOrderStorageTimeExpired: {codes.InvalidArgument, "storage time expired"},
	module.ErrRecipientNotFound:       {codes.NotFound, "recipient not found"},
	module.ErrOrdersDifferentClients:  {codes.InvalidArgument, "order does not exists"},
	module.ErrOrderNotIssuedOrExpired: {codes.InvalidArgument, "order not issued or expired"},
	module.ErrOrderNotExpiredOrIssued: {codes.InvalidArgument, "order issued or not expired"},
	domain.ErrPackageTypeUnsupported:  {codes.InvalidArgument, "invalid package type"},
	domain.ErrWeightNegative:          {codes.InvalidArgument, "invalid weight"},
}

func handleOrderError(err error) error {
	for key, value := range errorMap {
		if errors.Is(err, key) {
			return status.Errorf(value.code, "%s: %v", value.message, err)
		}
	}
	return status.Errorf(codes.Internal, "internal server error: %v", err)
}
