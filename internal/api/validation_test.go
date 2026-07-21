package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/damakalshchikov/test-task-junior-golang-developer/internal/models"
)

const testUserID = "11111111-1111-1111-1111-111111111111"

func TestValidateBodyRejectsInvalidPayloads(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantError string
	}{
		{
			name:      "missing service_name",
			body:      `{"price":100,"user_id":"` + testUserID + `","start_date":"01-2024"}`,
			wantError: "service_name is required",
		},
		{
			name:      "blank service_name",
			body:      `{"service_name":"","price":100,"user_id":"` + testUserID + `","start_date":"01-2024"}`,
			wantError: "service_name is required",
		},
		{
			name:      "missing price",
			body:      `{"service_name":"Netflix","user_id":"` + testUserID + `","start_date":"01-2024"}`,
			wantError: "price is required",
		},
		{
			name:      "negative price",
			body:      `{"service_name":"Netflix","price":-1,"user_id":"` + testUserID + `","start_date":"01-2024"}`,
			wantError: "price must be greater than or equal to 0",
		},
		{
			name:      "missing user_id",
			body:      `{"service_name":"Netflix","price":100,"start_date":"01-2024"}`,
			wantError: "user_id is required",
		},
		{
			name:      "nil user_id",
			body:      `{"service_name":"Netflix","price":100,"user_id":"00000000-0000-0000-0000-000000000000","start_date":"01-2024"}`,
			wantError: "user_id is required",
		},
		{
			name:      "missing start_date",
			body:      `{"service_name":"Netflix","price":100,"user_id":"` + testUserID + `"}`,
			wantError: "start_date is required",
		},
		{
			name:      "end_date before start_date",
			body:      `{"service_name":"Netflix","price":100,"user_id":"` + testUserID + `","start_date":"03-2024","end_date":"01-2024"}`,
			wantError: "end_date must not be before start_date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec, called := runValidateBody(t, tt.body)

			if called {
				t.Error("next handler was called for an invalid payload")
			}

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}

			if got := decodeErrorBody(t, rec); got != tt.wantError {
				t.Errorf("error = %q, want %q", got, tt.wantError)
			}
		})
	}
}

func TestValidateBodyRejectsMalformedJSON(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"broken json", `{"service_name":`},
		{"empty body", ``},
		{"wrong date format", `{"service_name":"Netflix","price":100,"user_id":"` + testUserID + `","start_date":"2024-01"}`},
		{"date out of range", `{"service_name":"Netflix","price":100,"user_id":"` + testUserID + `","start_date":"13-2024"}`},
		{"price is a string", `{"service_name":"Netflix","price":"100","user_id":"` + testUserID + `","start_date":"01-2024"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec, called := runValidateBody(t, tt.body)

			if called {
				t.Error("next handler was called for a malformed payload")
			}

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}

			if got := decodeErrorBody(t, rec); !strings.HasPrefix(got, "invalid request body") {
				t.Errorf("error = %q, want prefix %q", got, "invalid request body")
			}
		})
	}
}

func TestValidateBodyAcceptsValidPayloads(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"full payload", `{"service_name":"Netflix","price":100,"user_id":"` + testUserID + `","start_date":"01-2024","end_date":"03-2024"}`},
		{"without end_date", `{"service_name":"Netflix","price":100,"user_id":"` + testUserID + `","start_date":"01-2024"}`},
		{"zero price", `{"service_name":"Netflix","price":0,"user_id":"` + testUserID + `","start_date":"01-2024"}`},
		{"end_date equals start_date", `{"service_name":"Netflix","price":100,"user_id":"` + testUserID + `","start_date":"01-2024","end_date":"01-2024"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec, called := runValidateBody(t, tt.body)

			if !called {
				t.Fatalf("next handler was not called, status = %d, body = %s", rec.Code, rec.Body)
			}

			if rec.Code != http.StatusOK {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
			}
		})
	}
}

func TestValidateBodyPassesDecodedBodyThroughContext(t *testing.T) {
	body := `{"service_name":"Netflix","price":100,"user_id":"` + testUserID + `","start_date":"01-2024","end_date":"03-2024"}`

	var got models.SubscriptionRequest
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = bodyFrom[models.SubscriptionRequest](r.Context())
	})

	req := httptest.NewRequest(http.MethodPost, "/subscriptions", strings.NewReader(body))
	validateBody[models.SubscriptionRequest](next).ServeHTTP(httptest.NewRecorder(), req)

	if got.ServiceName != "Netflix" {
		t.Errorf("service_name = %q, want %q", got.ServiceName, "Netflix")
	}

	if got.Price == nil || *got.Price != 100 {
		t.Errorf("price = %v, want 100", got.Price)
	}

	if got.UserID.String() != testUserID {
		t.Errorf("user_id = %v, want %v", got.UserID, testUserID)
	}

	if got.StartDate.Format("01-2006") != "01-2024" {
		t.Errorf("start_date = %v, want 01-2024", got.StartDate.Time)
	}

	if got.EndDate == nil || got.EndDate.Format("01-2006") != "03-2024" {
		t.Errorf("end_date = %v, want 03-2024", got.EndDate)
	}
}

func runValidateBody(t *testing.T, body string) (*httptest.ResponseRecorder, bool) {
	t.Helper()

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/subscriptions", strings.NewReader(body))
	rec := httptest.NewRecorder()

	validateBody[models.SubscriptionRequest](next).ServeHTTP(rec, req)

	return rec, called
}

func decodeErrorBody(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error response: %v", err)
	}

	return resp.Error
}
