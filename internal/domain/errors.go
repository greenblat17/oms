package domain

import (
	"errors"
	"fmt"
)

var (
	ErrPackageTypeUnsupported = errors.New("unsupported package type")
	ErrWeightNegative         = errors.New("weight is negative")
)

type ErrWeightExceedsLimit struct {
	Weight float64
	Limit  float64
}

func (e ErrWeightExceedsLimit) Error() string {
	return fmt.Sprintf("weight exceeds %f kg limit: got %f kg", e.Limit, e.Weight)
}
