package db

import "time"

// Owner types, stored in owner_type columns.
const (
	OwnerCharacter   = "character"
	OwnerCorporation = "corporation"
)

// nullIfZero turns the zero value ESI uses for a missing optional field into
// a NULL.
func nullIfZero[T comparable](v T) any {
	var zero T
	if v == zero {
		return nil
	}
	return v
}

func nullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}
