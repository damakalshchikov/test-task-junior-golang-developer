package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/damakalshchikov/test-task-junior-golang-developer/internal/models"
	"github.com/damakalshchikov/test-task-junior-golang-developer/internal/storage"
)

const (
	testSubscriptionID = "22222222-2222-2222-2222-222222222222"
	validBody          = `{"service_name":"Netflix","price":100,"user_id":"` + testUserID + `","start_date":"01-2024","end_date":"03-2024"}`
)

var errStorageFailure = errors.New("connection refused")

type fakeStorage struct {
	createFn    func(ctx context.Context, sub *models.Subscription) error
	getByIDFn   func(ctx context.Context, id uuid.UUID) (*models.Subscription, error)
	listFn      func(ctx context.Context, filter storage.ListFilter) ([]models.Subscription, error)
	updateFn    func(ctx context.Context, sub *models.Subscription) error
	deleteFn    func(ctx context.Context, id uuid.UUID) error
	totalCostFn func(ctx context.Context, filter storage.SummaryFilter) (int, error)
}

func (f *fakeStorage) Create(ctx context.Context, sub *models.Subscription) error {
	return f.createFn(ctx, sub)
}

func (f *fakeStorage) GetByID(ctx context.Context, id uuid.UUID) (*models.Subscription, error) {
	return f.getByIDFn(ctx, id)
}

func (f *fakeStorage) List(ctx context.Context, filter storage.ListFilter) ([]models.Subscription, error) {
	return f.listFn(ctx, filter)
}

func (f *fakeStorage) Update(ctx context.Context, sub *models.Subscription) error {
	return f.updateFn(ctx, sub)
}

func (f *fakeStorage) Delete(ctx context.Context, id uuid.UUID) error {
	return f.deleteFn(ctx, id)
}

func (f *fakeStorage) TotalCost(ctx context.Context, filter storage.SummaryFilter) (int, error) {
	return f.totalCostFn(ctx, filter)
}

func newTestRouter(st SubscriptionStorage) http.Handler {
	return NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), st)
}

func doRequest(t *testing.T, router http.Handler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(method, target, reader))

	return rec
}

func sampleSubscription() *models.Subscription {
	return &models.Subscription{
		ID:          uuid.MustParse(testSubscriptionID),
		ServiceName: "Netflix",
		Price:       100,
		UserID:      uuid.MustParse(testUserID),
		StartDate:   models.MonthYear{Time: time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)},
		CreatedAt:   time.Date(2024, time.January, 10, 12, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2024, time.January, 10, 12, 0, 0, 0, time.UTC),
	}
}

func TestHealth(t *testing.T) {
	rec := doRequest(t, newTestRouter(&fakeStorage{}), http.MethodGet, "/health", "")

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestCreateSubscription(t *testing.T) {
	var received *models.Subscription

	router := newTestRouter(&fakeStorage{
		createFn: func(ctx context.Context, sub *models.Subscription) error {
			received = sub
			sub.ID = uuid.MustParse(testSubscriptionID)
			return nil
		},
	})

	rec := doRequest(t, router, http.MethodPost, "/subscriptions", validBody)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body)
	}

	if received == nil {
		t.Fatal("storage.Create was not called")
	}

	if received.ServiceName != "Netflix" || received.Price != 100 {
		t.Errorf("storage received %+v", received)
	}

	if received.UserID.String() != testUserID {
		t.Errorf("user_id = %v, want %v", received.UserID, testUserID)
	}

	var got models.Subscription
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got.ID.String() != testSubscriptionID {
		t.Errorf("response id = %v, want %v", got.ID, testSubscriptionID)
	}
}

func TestCreateSubscriptionInvalidBody(t *testing.T) {
	router := newTestRouter(&fakeStorage{
		createFn: func(ctx context.Context, sub *models.Subscription) error {
			t.Error("storage.Create must not be called for an invalid body")
			return nil
		},
	})

	rec := doRequest(t, router, http.MethodPost, "/subscriptions", `{"service_name":"Netflix"}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCreateSubscriptionStorageError(t *testing.T) {
	router := newTestRouter(&fakeStorage{
		createFn: func(ctx context.Context, sub *models.Subscription) error {
			return errStorageFailure
		},
	})

	rec := doRequest(t, router, http.MethodPost, "/subscriptions", validBody)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	got := decodeErrorBody(t, rec)
	if got != "internal server error" {
		t.Errorf("error = %q, want %q", got, "internal server error")
	}

	if strings.Contains(got, errStorageFailure.Error()) {
		t.Error("internal error details leaked to the client")
	}
}

func TestGetSubscriptionByID(t *testing.T) {
	var requested uuid.UUID

	router := newTestRouter(&fakeStorage{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Subscription, error) {
			requested = id
			return sampleSubscription(), nil
		},
	})

	rec := doRequest(t, router, http.MethodGet, "/subscriptions/"+testSubscriptionID, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if requested.String() != testSubscriptionID {
		t.Errorf("storage received id = %v, want %v", requested, testSubscriptionID)
	}

	var got models.Subscription
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got.ServiceName != "Netflix" {
		t.Errorf("service_name = %q, want %q", got.ServiceName, "Netflix")
	}
}

func TestGetSubscriptionByIDNotFound(t *testing.T) {
	router := newTestRouter(&fakeStorage{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Subscription, error) {
			return nil, storage.ErrNotFound
		},
	})

	rec := doRequest(t, router, http.MethodGet, "/subscriptions/"+testSubscriptionID, "")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	if got := decodeErrorBody(t, rec); got != "subscription not found" {
		t.Errorf("error = %q, want %q", got, "subscription not found")
	}
}

func TestSubscriptionInvalidID(t *testing.T) {
	fail := func(name string) func(ctx context.Context, id uuid.UUID) (*models.Subscription, error) {
		return func(ctx context.Context, id uuid.UUID) (*models.Subscription, error) {
			t.Errorf("%s must not reach the storage", name)
			return nil, nil
		}
	}

	router := newTestRouter(&fakeStorage{
		getByIDFn: fail("GET"),
		updateFn: func(ctx context.Context, sub *models.Subscription) error {
			t.Error("PUT must not reach the storage")
			return nil
		},
		deleteFn: func(ctx context.Context, id uuid.UUID) error {
			t.Error("DELETE must not reach the storage")
			return nil
		},
	})

	tests := []struct {
		method string
		body   string
	}{
		{http.MethodGet, ""},
		{http.MethodPut, validBody},
		{http.MethodDelete, ""},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			rec := doRequest(t, router, tt.method, "/subscriptions/not-a-uuid", tt.body)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}

			if got := decodeErrorBody(t, rec); got != "id must be a valid UUID" {
				t.Errorf("error = %q, want %q", got, "id must be a valid UUID")
			}
		})
	}
}

func TestListSubscriptionsDefaultFilter(t *testing.T) {
	var received storage.ListFilter

	router := newTestRouter(&fakeStorage{
		listFn: func(ctx context.Context, filter storage.ListFilter) ([]models.Subscription, error) {
			received = filter
			return []models.Subscription{*sampleSubscription()}, nil
		},
	})

	rec := doRequest(t, router, http.MethodGet, "/subscriptions", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if received.Limit != 50 {
		t.Errorf("limit = %d, want 50", received.Limit)
	}

	if received.Offset != 0 {
		t.Errorf("offset = %d, want 0", received.Offset)
	}

	if received.UserID != nil || received.ServiceName != nil {
		t.Errorf("filters = %+v, want both nil", received)
	}
}

func TestListSubscriptionsWithFilters(t *testing.T) {
	var received storage.ListFilter

	router := newTestRouter(&fakeStorage{
		listFn: func(ctx context.Context, filter storage.ListFilter) ([]models.Subscription, error) {
			received = filter
			return []models.Subscription{}, nil
		},
	})

	target := "/subscriptions?user_id=" + testUserID + "&service_name=Netflix&limit=10&offset=20"
	rec := doRequest(t, router, http.MethodGet, target, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if received.UserID == nil || received.UserID.String() != testUserID {
		t.Errorf("user_id = %v, want %v", received.UserID, testUserID)
	}

	if received.ServiceName == nil || *received.ServiceName != "Netflix" {
		t.Errorf("service_name = %v, want Netflix", received.ServiceName)
	}

	if received.Limit != 10 || received.Offset != 20 {
		t.Errorf("limit/offset = %d/%d, want 10/20", received.Limit, received.Offset)
	}
}

func TestListSubscriptionsInvalidQuery(t *testing.T) {
	router := newTestRouter(&fakeStorage{
		listFn: func(ctx context.Context, filter storage.ListFilter) ([]models.Subscription, error) {
			t.Error("storage.List must not be called for an invalid query")
			return nil, nil
		},
	})

	tests := []struct {
		name      string
		query     string
		wantError string
	}{
		{"bad user_id", "?user_id=not-a-uuid", "user_id must be a valid UUID"},
		{"limit below range", "?limit=0", "limit must be an integer between 1 and 100"},
		{"limit above range", "?limit=101", "limit must be an integer between 1 and 100"},
		{"limit not a number", "?limit=abc", "limit must be an integer between 1 and 100"},
		{"negative offset", "?offset=-1", "offset must be a non-negative integer"},
		{"offset not a number", "?offset=abc", "offset must be a non-negative integer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := doRequest(t, router, http.MethodGet, "/subscriptions"+tt.query, "")

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}

			if got := decodeErrorBody(t, rec); got != tt.wantError {
				t.Errorf("error = %q, want %q", got, tt.wantError)
			}
		})
	}
}

func TestListSubscriptionsEmptyResultIsArray(t *testing.T) {
	router := newTestRouter(&fakeStorage{
		listFn: func(ctx context.Context, filter storage.ListFilter) ([]models.Subscription, error) {
			return []models.Subscription{}, nil
		},
	})

	rec := doRequest(t, router, http.MethodGet, "/subscriptions", "")

	if body := strings.TrimSpace(rec.Body.String()); body != "[]" {
		t.Errorf("body = %s, want []", body)
	}
}

func TestUpdateSubscription(t *testing.T) {
	var received *models.Subscription

	router := newTestRouter(&fakeStorage{
		updateFn: func(ctx context.Context, sub *models.Subscription) error {
			received = sub
			return nil
		},
	})

	rec := doRequest(t, router, http.MethodPut, "/subscriptions/"+testSubscriptionID, validBody)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body)
	}

	if received == nil {
		t.Fatal("storage.Update was not called")
	}

	if received.ID.String() != testSubscriptionID {
		t.Errorf("id = %v, want %v", received.ID, testSubscriptionID)
	}

	if received.ServiceName != "Netflix" {
		t.Errorf("service_name = %q, want %q", received.ServiceName, "Netflix")
	}
}

func TestUpdateSubscriptionNotFound(t *testing.T) {
	router := newTestRouter(&fakeStorage{
		updateFn: func(ctx context.Context, sub *models.Subscription) error {
			return storage.ErrNotFound
		},
	})

	rec := doRequest(t, router, http.MethodPut, "/subscriptions/"+testSubscriptionID, validBody)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestUpdateSubscriptionInvalidBody(t *testing.T) {
	router := newTestRouter(&fakeStorage{
		updateFn: func(ctx context.Context, sub *models.Subscription) error {
			t.Error("storage.Update must not be called for an invalid body")
			return nil
		},
	})

	rec := doRequest(t, router, http.MethodPut, "/subscriptions/"+testSubscriptionID, `{"price":-5}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestDeleteSubscription(t *testing.T) {
	var requested uuid.UUID

	router := newTestRouter(&fakeStorage{
		deleteFn: func(ctx context.Context, id uuid.UUID) error {
			requested = id
			return nil
		},
	})

	rec := doRequest(t, router, http.MethodDelete, "/subscriptions/"+testSubscriptionID, "")

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}

	if requested.String() != testSubscriptionID {
		t.Errorf("storage received id = %v, want %v", requested, testSubscriptionID)
	}

	if rec.Body.Len() != 0 {
		t.Errorf("body = %s, want empty", rec.Body)
	}
}

func TestDeleteSubscriptionNotFound(t *testing.T) {
	router := newTestRouter(&fakeStorage{
		deleteFn: func(ctx context.Context, id uuid.UUID) error {
			return storage.ErrNotFound
		},
	})

	rec := doRequest(t, router, http.MethodDelete, "/subscriptions/"+testSubscriptionID, "")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestSummary(t *testing.T) {
	var received storage.SummaryFilter

	router := newTestRouter(&fakeStorage{
		totalCostFn: func(ctx context.Context, filter storage.SummaryFilter) (int, error) {
			received = filter
			return 4650, nil
		},
	})

	target := "/subscriptions/summary?from=01-2024&to=12-2024&user_id=" + testUserID + "&service_name=Netflix"
	rec := doRequest(t, router, http.MethodGet, target, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body)
	}

	wantFrom := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	wantTo := time.Date(2024, time.December, 1, 0, 0, 0, 0, time.UTC)

	if !received.From.Equal(wantFrom) || !received.To.Equal(wantTo) {
		t.Errorf("period = %v..%v, want %v..%v", received.From, received.To, wantFrom, wantTo)
	}

	if received.UserID == nil || received.UserID.String() != testUserID {
		t.Errorf("user_id = %v, want %v", received.UserID, testUserID)
	}

	if received.ServiceName == nil || *received.ServiceName != "Netflix" {
		t.Errorf("service_name = %v, want Netflix", received.ServiceName)
	}

	var got summaryResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got.TotalCost != 4650 {
		t.Errorf("total_cost = %d, want 4650", got.TotalCost)
	}

	if got.From != "01-2024" || got.To != "12-2024" {
		t.Errorf("period in response = %s..%s, want 01-2024..12-2024", got.From, got.To)
	}
}

func TestSummaryWithoutOptionalFilters(t *testing.T) {
	var received storage.SummaryFilter

	router := newTestRouter(&fakeStorage{
		totalCostFn: func(ctx context.Context, filter storage.SummaryFilter) (int, error) {
			received = filter
			return 0, nil
		},
	})

	rec := doRequest(t, router, http.MethodGet, "/subscriptions/summary?from=01-2024&to=01-2024", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if received.UserID != nil || received.ServiceName != nil {
		t.Errorf("filters = %+v, want both nil", received)
	}
}

func TestSummaryInvalidQuery(t *testing.T) {
	router := newTestRouter(&fakeStorage{
		totalCostFn: func(ctx context.Context, filter storage.SummaryFilter) (int, error) {
			t.Error("storage.TotalCost must not be called for an invalid query")
			return 0, nil
		},
	})

	tests := []struct {
		name      string
		query     string
		wantError string
	}{
		{"missing both", "", "from is required, format MM-YYYY"},
		{"missing from", "?to=12-2024", "from is required, format MM-YYYY"},
		{"missing to", "?from=01-2024", "to is required, format MM-YYYY"},
		{"bad from format", "?from=2024-01&to=12-2024", "from is required, format MM-YYYY"},
		{"bad to format", "?from=01-2024&to=2024-12", "to is required, format MM-YYYY"},
		{"to before from", "?from=12-2024&to=01-2024", "to must not be before from"},
		{"bad user_id", "?from=01-2024&to=12-2024&user_id=not-a-uuid", "user_id must be a valid UUID"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := doRequest(t, router, http.MethodGet, "/subscriptions/summary"+tt.query, "")

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}

			if got := decodeErrorBody(t, rec); got != tt.wantError {
				t.Errorf("error = %q, want %q", got, tt.wantError)
			}
		})
	}
}

func TestSummaryStorageError(t *testing.T) {
	router := newTestRouter(&fakeStorage{
		totalCostFn: func(ctx context.Context, filter storage.SummaryFilter) (int, error) {
			return 0, errStorageFailure
		},
	})

	rec := doRequest(t, router, http.MethodGet, "/subscriptions/summary?from=01-2024&to=12-2024", "")

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	if got := decodeErrorBody(t, rec); got != "internal server error" {
		t.Errorf("error = %q, want %q", got, "internal server error")
	}
}
