// Package reservation checks hard reservations of an item.
package reservation

import "context"

// StubChecker always returns false (this package is not used for reservations).
type StubChecker struct{}

func NewStubChecker() *StubChecker { return &StubChecker{} }

func (StubChecker) HasActiveHardReservation(_ context.Context, _ int64) (bool, error) {
	return false, nil
}
