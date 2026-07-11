package models

import (
	"fmt"
	"strings"
	"time"
)

const monthYearLayout = "01-2006"

type MonthYear struct {
	time.Time
}

func (m MonthYear) MarshalJSON() ([]byte, error) {
	return []byte(`"` + m.Format(monthYearLayout) + `"`), nil
}

func (m *MonthYear) UnmarshalJSON(data []byte) error {
	value := strings.Trim(string(data), `"`)

	parsed, err := time.Parse(monthYearLayout, value)
	if err != nil {
		return fmt.Errorf("invalid date %q, expected format MM-YYYY", value)
	}

	m.Time = parsed

	return nil
}
