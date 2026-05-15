package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type JSONStringArray []string

func (a JSONStringArray) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *JSONStringArray) Scan(value any) error {
	if value == nil {
		*a = JSONStringArray{}
		return nil
	}

	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, a)
}
