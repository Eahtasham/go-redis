package store

import "time"

type Entry struct {
	Type   ValueType
	Value  any
	Expiry time.Time
}

func (e *Entry) IsExpired() bool {
	if e.Expiry.IsZero() {
		return false
	}

	return time.Now().After(e.Expiry)
}
