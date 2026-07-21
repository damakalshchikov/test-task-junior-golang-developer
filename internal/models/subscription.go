package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type Subscription struct {
	ID          uuid.UUID  `json:"id"`
	ServiceName string     `json:"service_name"`
	Price       int        `json:"price"`
	UserID      uuid.UUID  `json:"user_id"`
	StartDate   MonthYear  `json:"start_date"`
	EndDate     *MonthYear `json:"end_date,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type SubscriptionRequest struct {
	ServiceName string     `json:"service_name" validate:"required"`
	Price       *int       `json:"price" validate:"required,min=0"`
	UserID      uuid.UUID  `json:"user_id" validate:"required"`
	StartDate   MonthYear  `json:"start_date" validate:"required"`
	EndDate     *MonthYear `json:"end_date,omitempty" validate:"omitempty,gtefield=StartDate"`
}

func (r SubscriptionRequest) ToSubscription() *Subscription {
	return &Subscription{
		ServiceName: strings.TrimSpace(r.ServiceName),
		Price:       *r.Price,
		UserID:      r.UserID,
		StartDate:   r.StartDate,
		EndDate:     r.EndDate,
	}
}
