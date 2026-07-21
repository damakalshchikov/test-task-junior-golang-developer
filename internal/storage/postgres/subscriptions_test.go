package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/damakalshchikov/test-task-junior-golang-developer/internal/models"
	"github.com/damakalshchikov/test-task-junior-golang-developer/internal/storage"
)

func newSubscription(t *testing.T, service string, price int, userID uuid.UUID, start, end string) *models.Subscription {
	t.Helper()

	sub := &models.Subscription{
		ServiceName: service,
		Price:       price,
		UserID:      userID,
		StartDate:   monthYear(t, start),
	}

	if end != "" {
		sub.EndDate = monthYearPtr(t, end)
	}

	return sub
}

func mustCreate(t *testing.T, st *SubscriptionStorage, sub *models.Subscription) *models.Subscription {
	t.Helper()

	if err := st.Create(context.Background(), sub); err != nil {
		t.Fatalf("create subscription: %v", err)
	}

	return sub
}

func TestCreateAndGetByID(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	userID := uuid.New()

	created := mustCreate(t, st, newSubscription(t, "Yandex Plus", 400, userID, "01-2024", "06-2024"))

	if created.ID == uuid.Nil {
		t.Fatal("id was not populated after create")
	}

	if created.CreatedAt.IsZero() || created.UpdatedAt.IsZero() {
		t.Error("timestamps were not populated after create")
	}

	got, err := st.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}

	if got.ServiceName != "Yandex Plus" || got.Price != 400 || got.UserID != userID {
		t.Errorf("got %+v, want service=Yandex Plus price=400 user=%v", got, userID)
	}

	if !got.StartDate.Time.Equal(monthYear(t, "01-2024").Time) {
		t.Errorf("start_date = %v, want 01-2024", got.StartDate.Time)
	}

	if got.EndDate == nil || !got.EndDate.Time.Equal(monthYear(t, "06-2024").Time) {
		t.Errorf("end_date = %v, want 06-2024", got.EndDate)
	}
}

func TestCreateWithoutEndDate(t *testing.T) {
	st := newTestStorage(t)

	created := mustCreate(t, st, newSubscription(t, "Netflix", 100, uuid.New(), "01-2024", ""))

	got, err := st.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}

	if got.EndDate != nil {
		t.Errorf("end_date = %v, want nil", got.EndDate.Time)
	}
}

func TestGetByIDNotFound(t *testing.T) {
	st := newTestStorage(t)

	_, err := st.GetByID(context.Background(), uuid.New())
	if !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("error = %v, want %v", err, storage.ErrNotFound)
	}
}

func TestUpdateSubscription(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	created := mustCreate(t, st, newSubscription(t, "Netflix", 100, uuid.New(), "01-2024", "06-2024"))

	updated := newSubscription(t, "Spotify", 250, uuid.New(), "02-2024", "")
	updated.ID = created.ID

	if err := st.Update(ctx, updated); err != nil {
		t.Fatalf("update: %v", err)
	}

	got, err := st.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}

	if got.ServiceName != "Spotify" || got.Price != 250 || got.UserID != updated.UserID {
		t.Errorf("got %+v, want service=Spotify price=250", got)
	}

	if got.EndDate != nil {
		t.Errorf("end_date = %v, want nil after update", got.EndDate.Time)
	}

	if !got.UpdatedAt.After(got.CreatedAt) {
		t.Errorf("updated_at %v is not after created_at %v", got.UpdatedAt, got.CreatedAt)
	}
}

func TestUpdateSubscriptionNotFound(t *testing.T) {
	st := newTestStorage(t)

	sub := newSubscription(t, "Netflix", 100, uuid.New(), "01-2024", "")
	sub.ID = uuid.New()

	if err := st.Update(context.Background(), sub); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("error = %v, want %v", err, storage.ErrNotFound)
	}
}

func TestDeleteSubscription(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	created := mustCreate(t, st, newSubscription(t, "Netflix", 100, uuid.New(), "01-2024", ""))

	if err := st.Delete(ctx, created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if _, err := st.GetByID(ctx, created.ID); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("error after delete = %v, want %v", err, storage.ErrNotFound)
	}
}

func TestDeleteSubscriptionNotFound(t *testing.T) {
	st := newTestStorage(t)

	if err := st.Delete(context.Background(), uuid.New()); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("error = %v, want %v", err, storage.ErrNotFound)
	}
}

func TestListReturnsEmptySliceNotNil(t *testing.T) {
	st := newTestStorage(t)

	subs, err := st.List(context.Background(), storage.ListFilter{Limit: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if subs == nil {
		t.Fatal("list returned nil, want empty slice")
	}

	if len(subs) != 0 {
		t.Errorf("len = %d, want 0", len(subs))
	}
}

func TestListOrderingAndPagination(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	userID := uuid.New()

	first := mustCreate(t, st, newSubscription(t, "First", 100, userID, "01-2024", ""))
	second := mustCreate(t, st, newSubscription(t, "Second", 200, userID, "02-2024", ""))
	third := mustCreate(t, st, newSubscription(t, "Third", 300, userID, "03-2024", ""))

	subs, err := st.List(ctx, storage.ListFilter{Limit: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	wantOrder := []uuid.UUID{third.ID, second.ID, first.ID}
	if len(subs) != len(wantOrder) {
		t.Fatalf("len = %d, want %d", len(subs), len(wantOrder))
	}

	for i, want := range wantOrder {
		if subs[i].ID != want {
			t.Errorf("position %d = %v, want %v", i, subs[i].ID, want)
		}
	}

	page, err := st.List(ctx, storage.ListFilter{Limit: 2, Offset: 2})
	if err != nil {
		t.Fatalf("list with offset: %v", err)
	}

	if len(page) != 1 || page[0].ID != first.ID {
		t.Errorf("page = %+v, want only %v", page, first.ID)
	}
}

func TestListFilters(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	userA := uuid.New()
	userB := uuid.New()

	mustCreate(t, st, newSubscription(t, "Netflix", 100, userA, "01-2024", ""))
	mustCreate(t, st, newSubscription(t, "Spotify", 200, userA, "01-2024", ""))
	mustCreate(t, st, newSubscription(t, "Netflix", 300, userB, "01-2024", ""))

	netflix := "Netflix"

	tests := []struct {
		name   string
		filter storage.ListFilter
		want   int
	}{
		{"no filters", storage.ListFilter{Limit: 10}, 3},
		{"by user", storage.ListFilter{Limit: 10, UserID: &userA}, 2},
		{"by service", storage.ListFilter{Limit: 10, ServiceName: &netflix}, 2},
		{"by user and service", storage.ListFilter{Limit: 10, UserID: &userA, ServiceName: &netflix}, 1},
		{"limit applied", storage.ListFilter{Limit: 1}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subs, err := st.List(ctx, tt.filter)
			if err != nil {
				t.Fatalf("list: %v", err)
			}

			if len(subs) != tt.want {
				t.Errorf("len = %d, want %d", len(subs), tt.want)
			}
		})
	}
}

func TestTotalCost(t *testing.T) {
	userA := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	userB := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

	netflix := "Netflix"

	type fixture struct {
		service string
		price   int
		user    uuid.UUID
		start   string
		end     string
	}

	tests := []struct {
		name        string
		fixtures    []fixture
		from        string
		to          string
		userID      *uuid.UUID
		serviceName *string
		want        int
	}{
		{
			name:     "no subscriptions",
			from:     "01-2024",
			to:       "12-2024",
			want:     0,
			fixtures: nil,
		},
		{
			name:     "subscription inside period",
			fixtures: []fixture{{"Netflix", 100, userA, "02-2024", "04-2024"}},
			from:     "01-2024", to: "12-2024",
			want: 300,
		},
		{
			name:     "subscription overlaps period start",
			fixtures: []fixture{{"Netflix", 100, userA, "11-2023", "02-2024"}},
			from:     "01-2024", to: "12-2024",
			want: 200,
		},
		{
			name:     "subscription overlaps period end",
			fixtures: []fixture{{"Netflix", 100, userA, "11-2024", "03-2025"}},
			from:     "01-2024", to: "12-2024",
			want: 200,
		},
		{
			name:     "subscription covers whole period",
			fixtures: []fixture{{"Netflix", 100, userA, "01-2023", "12-2025"}},
			from:     "01-2024", to: "12-2024",
			want: 1200,
		},
		{
			name:     "subscription entirely before period",
			fixtures: []fixture{{"Netflix", 100, userA, "01-2023", "06-2023"}},
			from:     "01-2024", to: "12-2024",
			want: 0,
		},
		{
			name:     "subscription entirely after period",
			fixtures: []fixture{{"Netflix", 100, userA, "01-2025", "06-2025"}},
			from:     "01-2024", to: "12-2024",
			want: 0,
		},
		{
			name:     "subscription ends exactly at period start",
			fixtures: []fixture{{"Netflix", 100, userA, "01-2023", "01-2024"}},
			from:     "01-2024", to: "12-2024",
			want: 100,
		},
		{
			name:     "subscription starts exactly at period end",
			fixtures: []fixture{{"Netflix", 100, userA, "12-2024", "06-2025"}},
			from:     "01-2024", to: "12-2024",
			want: 100,
		},
		{
			name:     "open ended subscription starting inside period",
			fixtures: []fixture{{"Netflix", 100, userA, "06-2024", ""}},
			from:     "01-2024", to: "12-2024",
			want: 700,
		},
		{
			name:     "open ended subscription starting before period",
			fixtures: []fixture{{"Netflix", 100, userA, "01-2023", ""}},
			from:     "01-2024", to: "12-2024",
			want: 1200,
		},
		{
			name:     "single month period",
			fixtures: []fixture{{"Netflix", 100, userA, "01-2024", "12-2024"}},
			from:     "03-2024", to: "03-2024",
			want: 100,
		},
		{
			name: "sums several subscriptions",
			fixtures: []fixture{
				{"Netflix", 100, userA, "01-2024", "12-2024"},
				{"Spotify", 200, userA, "01-2024", "06-2024"},
			},
			from: "01-2024", to: "12-2024",
			want: 2400,
		},
		{
			name: "filters by user",
			fixtures: []fixture{
				{"Netflix", 100, userA, "01-2024", "12-2024"},
				{"Netflix", 100, userB, "01-2024", "12-2024"},
			},
			from: "01-2024", to: "12-2024",
			userID: &userA,
			want:   1200,
		},
		{
			name: "filters by service name",
			fixtures: []fixture{
				{"Netflix", 100, userA, "01-2024", "12-2024"},
				{"Spotify", 200, userA, "01-2024", "12-2024"},
			},
			from: "01-2024", to: "12-2024",
			serviceName: &netflix,
			want:        1200,
		},
		{
			name: "filters by user and service name",
			fixtures: []fixture{
				{"Netflix", 100, userA, "01-2024", "12-2024"},
				{"Netflix", 100, userB, "01-2024", "12-2024"},
				{"Spotify", 200, userA, "01-2024", "12-2024"},
			},
			from: "01-2024", to: "12-2024",
			userID: &userA, serviceName: &netflix,
			want: 1200,
		},
		{
			name:     "filter matches nothing",
			fixtures: []fixture{{"Netflix", 100, userA, "01-2024", "12-2024"}},
			from:     "01-2024", to: "12-2024",
			userID: &userB,
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := newTestStorage(t)
			ctx := context.Background()

			for _, f := range tt.fixtures {
				mustCreate(t, st, newSubscription(t, f.service, f.price, f.user, f.start, f.end))
			}

			got, err := st.TotalCost(ctx, storage.SummaryFilter{
				From:        monthYear(t, tt.from).Time,
				To:          monthYear(t, tt.to).Time,
				UserID:      tt.userID,
				ServiceName: tt.serviceName,
			})
			if err != nil {
				t.Fatalf("total cost: %v", err)
			}

			if got != tt.want {
				t.Errorf("total cost = %d, want %d", got, tt.want)
			}
		})
	}
}
