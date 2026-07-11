package storage

import (
	"errors"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("subscription not found")

type ListFilter struct {
	UserID      *uuid.UUID
	ServiceName *string
	Limit       int
	Offset      int
}
