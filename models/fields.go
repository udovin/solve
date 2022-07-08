package models

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// Cloner represents object that can be cloned.
type Cloner[T any] interface {
	Clone() T
}

// NInt64 represents nullable int64 with zero value means null value.
type NInt64 int64

// Value returns value.
func (v NInt64) Value() (driver.Value, error) {
	if v == 0 {
		return nil, nil
	}
	return int64(v), nil
}

// Scan scans value.
func (v *NInt64) Scan(value any) error {
	switch x := value.(type) {
	case nil:
		*v = 0
	case int64:
		*v = NInt64(x)
	default:
		return fmt.Errorf("unsupported type: %T", v)
	}
	return nil
}

var (
	_ driver.Valuer = NInt64(0)
	_ sql.Scanner   = (*NInt64)(nil)
)

// JSON represents json value.
type JSON []byte

const nullJSON = "null"

// Value returns value.
func (v JSON) Value() (driver.Value, error) {
	if len(v) == 0 {
		return nullJSON, nil
	}
	return string(v), nil
}

// Scan scans value.
func (v *JSON) Scan(value any) error {
	switch data := value.(type) {
	case nil:
		*v = nil
		return nil
	case []byte:
		return v.UnmarshalJSON(data)
	case string:
		return v.UnmarshalJSON([]byte(data))
	default:
		return fmt.Errorf("unsupported type: %T", data)
	}
}

// MarshalJSON marshals JSON.
func (v JSON) MarshalJSON() ([]byte, error) {
	if len(v) == 0 {
		return []byte(nullJSON), nil
	}
	return v, nil
}

// UnmarshalJSON unmarshals JSON.
func (v *JSON) UnmarshalJSON(bytes []byte) error {
	if !json.Valid(bytes) {
		return fmt.Errorf("invalid JSON value")
	}
	if string(bytes) == nullJSON {
		*v = nil
		return nil
	}
	*v = bytes
	return nil
}

func (v JSON) Clone() JSON {
	if v == nil {
		return nil
	}
	c := make(JSON, len(v))
	copy(c, v)
	return c
}

var (
	_ driver.Valuer = JSON{}
	_ sql.Scanner   = (*JSON)(nil)
)

// NString represents nullable string with empty value means null value.
type NString string

// Value returns value.
func (v NString) Value() (driver.Value, error) {
	if v == "" {
		return nil, nil
	}
	return string(v), nil
}

// Scan scans value.
func (v *NString) Scan(value any) error {
	switch x := value.(type) {
	case nil:
		*v = ""
	case string:
		*v = NString(x)
	case []byte:
		*v = NString(x)
	default:
		return fmt.Errorf("unsupported type: %T", v)
	}
	return nil
}

var (
	_ driver.Valuer = NString("")
	_ sql.Scanner   = (*NString)(nil)
)
