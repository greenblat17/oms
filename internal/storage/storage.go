package storage

import "errors"

var (
	ErrOrderNotFound   = errors.New("order not found")
	ErrOrderExists     = errors.New("order already exists")
	ErrOrderNotCreated = errors.New("order not created")
)
