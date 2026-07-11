package models

import (
	"errors"
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
	ServiceName string     `json:"service_name"`
	Price       *int       `json:"price"`
	UserID      uuid.UUID  `json:"user_id"`
	StartDate   MonthYear  `json:"start_date"`
	EndDate     *MonthYear `json:"end_date,omitempty"`
}

func (r SubscriptionRequest) Validate() error {
	if strings.TrimSpace(r.ServiceName) == "" {
		return errors.New("service_name is required")
	}

	if r.Price == nil {
		return errors.New("price is required")
	}

	if *r.Price < 0 {
		return errors.New("price must be non-negative")
	}

	if r.UserID == uuid.Nil {
		return errors.New("user_id is required")
	}

	if r.StartDate.IsZero() {
		return errors.New("start_date is required")
	}

	if r.EndDate != nil && r.EndDate.Before(r.StartDate.Time) {
		return errors.New("end_date must not be before start_date")
	}

	return nil
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
