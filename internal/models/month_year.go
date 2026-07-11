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

func ParseMonthYear(value string) (MonthYear, error) {
	parsed, err := time.Parse(monthYearLayout, value)
	if err != nil {
		return MonthYear{}, fmt.Errorf("invalid date %q, expected format MM-YYYY", value)
	}

	return MonthYear{Time: parsed}, nil
}

func (m MonthYear) MarshalJSON() ([]byte, error) {
	return []byte(`"` + m.Format(monthYearLayout) + `"`), nil
}

func (m *MonthYear) UnmarshalJSON(data []byte) error {
	parsed, err := ParseMonthYear(strings.Trim(string(data), `"`))
	if err != nil {
		return err
	}

	*m = parsed

	return nil
}
