package storage

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("subscription not found")

type ListFilter struct {
	UserID      *uuid.UUID
	ServiceName *string
	Limit       int
	Offset      int
}

type SummaryFilter struct {
	From        time.Time
	To          time.Time
	UserID      *uuid.UUID
	ServiceName *string
}
