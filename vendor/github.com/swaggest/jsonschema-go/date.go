package jsonschema

import (
	"encoding"
	"encoding/json"
	"time"
)

// DateLayout describes date format.
const DateLayout = "2006-01-02"

// Date is a date represented in YYYY-MM-DD format.
type Date time.Time

var (
	_ encoding.TextMarshaler   = new(Date)
	_ encoding.TextUnmarshaler = new(Date)
	_ json.Marshaler           = new(Date)
	_ json.Unmarshaler         = new(Date)
)

// UnmarshalText loads date from a standard format value.
func (d *Date) UnmarshalText(data []byte) error {
	t, err := time.Parse(DateLayout, string(data))
	if err != nil {
		return err
	}

	*d = Date(t)

	return nil
}

// MarshalText marshals date in standard format.
func (d Date) MarshalText() ([]byte, error) {
	return []byte(time.Time(d).Format(DateLayout)), nil
}

// UnmarshalJSON unmarshals date in standard format.
func (d *Date) UnmarshalJSON(data []byte) error {
	var s string

	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}

	return d.UnmarshalText([]byte(s))
}

// MarshalJSON marshals date in standard format.
func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(d).Format(DateLayout))
}
