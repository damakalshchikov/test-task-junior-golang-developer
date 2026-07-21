package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestToSubscription(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	price := 500
	start := MonthYear{Time: time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)}
	end := MonthYear{Time: time.Date(2024, time.March, 1, 0, 0, 0, 0, time.UTC)}

	req := SubscriptionRequest{
		ServiceName: "  Yandex Plus  ",
		Price:       &price,
		UserID:      userID,
		StartDate:   start,
		EndDate:     &end,
	}

	sub := req.ToSubscription()

	if sub.ServiceName != "Yandex Plus" {
		t.Errorf("service_name = %q, want %q", sub.ServiceName, "Yandex Plus")
	}

	if sub.Price != price {
		t.Errorf("price = %d, want %d", sub.Price, price)
	}

	if sub.UserID != userID {
		t.Errorf("user_id = %v, want %v", sub.UserID, userID)
	}

	if !sub.StartDate.Time.Equal(start.Time) {
		t.Errorf("start_date = %v, want %v", sub.StartDate.Time, start.Time)
	}

	if sub.EndDate == nil || !sub.EndDate.Time.Equal(end.Time) {
		t.Errorf("end_date = %v, want %v", sub.EndDate, end.Time)
	}
}

func TestToSubscriptionWithoutEndDate(t *testing.T) {
	price := 0

	req := SubscriptionRequest{
		ServiceName: "Netflix",
		Price:       &price,
		UserID:      uuid.New(),
		StartDate:   MonthYear{Time: time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)},
	}

	sub := req.ToSubscription()

	if sub.EndDate != nil {
		t.Errorf("end_date = %v, want nil", sub.EndDate.Time)
	}

	if sub.Price != 0 {
		t.Errorf("price = %d, want 0", sub.Price)
	}
}
