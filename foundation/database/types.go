package database

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

// ErrUnsupportedType is used when a scan attempt would have to unmarshal an unexpected type
var ErrUnsupportedType = errors.New("unsupported type")

// StringMap is a type used to make the go sql driver support some unsupported map types
type StringMap map[string]string

// Scan unmarshals value to Domain format
func (sm *StringMap) Scan(val interface{}) error {
	v, ok := val.([]byte)
	if !ok {
		return fmt.Errorf("type assertion failed, value is not a []byte: %w", ErrUnsupportedType)
	}

	if err := json.Unmarshal(v, &sm); err != nil {
		return fmt.Errorf("something went wrong while unmarshaling stringMap: %w", err)
	}

	return nil
}

// Value marshals value to DB format
func (sm *StringMap) Value() (driver.Value, error) {
	res, err := json.Marshal(sm)
	if err != nil {
		return nil, fmt.Errorf("something went wrong while parsing stringMap from database: %w", err)
	}

	return res, nil
}
