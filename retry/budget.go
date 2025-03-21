package retry

import (
	"sync"
	"time"
)

// Budget is an interface that defines methods for tracking and evaluating
// the rate of failures and successes in a retry scenario.
type Budget interface {
	// IsOver returns true if the rate of failures is over budget using the time provided.
	IsOver(now time.Time) bool
	// Failure records a number of failures for the time provided.
	Failure(now time.Time, hits int)
	// Attempt records a number of attempts for the time provided.
	Attempt(now time.Time, hits int)
	// Success records a number of successes for the time provided
	Success(now time.Time, hits int)
}

type budget struct {
	// TODO
	mutex sync.Mutex
}

// NewBudget creates a new Budget with the specified target failure rate.
// The returned budget is thread-safe and can be used as a global budget
// for limiting the total number of retries to a resource from an application,
// regardless of concurrent threads accessing the resource.
//
// 'ratio' is the maximum ratio of failures to successes allowed within the provided window.
// 'window' is the duration of the rolling window the budget is valid for
func NewBudget(ratio float64, window time.Duration) Budget {
	return &budget{}
}

// Failure records a number of failures for the given time.
// This method is thread-safe.
func (b *budget) Failure(now time.Time, hits int) {
	defer b.mutex.Unlock()
	b.mutex.Lock()
	// TODO
}

// Attempt records a number of attempts for the given time.
// This method is thread-safe.
func (b *budget) Attempt(now time.Time, hits int) {
	defer b.mutex.Unlock()
	b.mutex.Lock()
	// TODO
}

// Success records a number of attempts for the given time.
// This method is thread-safe.
func (b *budget) Success(now time.Time, hits int) {
	defer b.mutex.Unlock()
	b.mutex.Lock()
	// TODO
}

// IsOver determines if the current failure rate is over the budget.
// This method is thread-safe.
func (b *budget) IsOver(now time.Time) bool {
	defer b.mutex.Unlock()
	b.mutex.Lock()
	// TODO
	return false
}

// noOpBudget is a Budget implementation that always allows retries.
// It can be used when no budget control is desired.
type noOpBudget struct{}

// IsOver always returns false for noOpBudget, indicating that the budget is never exceeded.
func (noOpBudget) IsOver(now time.Time) bool {
	return false
}

// Failure is a no-op for noOpBudget.
func (noOpBudget) Failure(now time.Time, hits int) {}

// Attempt is a no-op for noOpBudget.
func (noOpBudget) Attempt(now time.Time, hits int) {}

// Success is a no-op for noOpBudget.
func (noOpBudget) Success(now time.Time, hits int) {}
