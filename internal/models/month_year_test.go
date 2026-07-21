package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseMonthYear(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{"january", "01-2024", time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC), false},
		{"december", "12-1999", time.Date(1999, time.December, 1, 0, 0, 0, 0, time.UTC), false},
		{"month above range", "13-2024", time.Time{}, true},
		{"zero month", "00-2024", time.Time{}, true},
		{"reversed order", "2024-01", time.Time{}, true},
		{"month without padding", "1-2024", time.Time{}, true},
		{"empty", "", time.Time{}, true},
		{"garbage", "abc", time.Time{}, true},
		{"extra text", "01-2024-15", time.Time{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseMonthYear(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseMonthYear(%q): expected error, got %v", tt.input, got.Time)
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseMonthYear(%q): unexpected error: %v", tt.input, err)
			}

			if !got.Time.Equal(tt.want) {
				t.Errorf("ParseMonthYear(%q) = %v, want %v", tt.input, got.Time, tt.want)
			}
		})
	}
}

func TestMonthYearMarshalJSON(t *testing.T) {
	value := MonthYear{Time: time.Date(2024, time.July, 1, 0, 0, 0, 0, time.UTC)}

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if string(data) != `"07-2024"` {
		t.Errorf("marshal = %s, want %q", data, `"07-2024"`)
	}
}

func TestMonthYearUnmarshalJSON(t *testing.T) {
	var value MonthYear

	if err := json.Unmarshal([]byte(`"07-2024"`), &value); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	want := time.Date(2024, time.July, 1, 0, 0, 0, 0, time.UTC)
	if !value.Time.Equal(want) {
		t.Errorf("unmarshal = %v, want %v", value.Time, want)
	}
}

func TestMonthYearUnmarshalJSONInvalid(t *testing.T) {
	for _, input := range []string{`"2024-07"`, `"13-2024"`, `""`, `"abc"`} {
		var value MonthYear

		if err := json.Unmarshal([]byte(input), &value); err == nil {
			t.Errorf("unmarshal(%s): expected error, got %v", input, value.Time)
		}
	}
}

func TestMonthYearRoundTrip(t *testing.T) {
	type payload struct {
		Start MonthYear  `json:"start_date"`
		End   *MonthYear `json:"end_date,omitempty"`
	}

	var decoded payload
	if err := json.Unmarshal([]byte(`{"start_date":"01-2024","end_date":"03-2024"}`), &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	encoded, err := json.Marshal(decoded)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	want := `{"start_date":"01-2024","end_date":"03-2024"}`
	if string(encoded) != want {
		t.Errorf("round trip = %s, want %s", encoded, want)
	}
}

func TestMonthYearOmittedEndDate(t *testing.T) {
	type payload struct {
		Start MonthYear  `json:"start_date"`
		End   *MonthYear `json:"end_date,omitempty"`
	}

	var decoded payload
	if err := json.Unmarshal([]byte(`{"start_date":"01-2024"}`), &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.End != nil {
		t.Errorf("end_date = %v, want nil", decoded.End.Time)
	}
}
