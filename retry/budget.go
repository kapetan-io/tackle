package retry

import (
	"sync"
	"time"
)

// Budget is an interface that defines methods for tracking and evaluating
// the rate of failures and successes in a retry scenario.
type Budget interface {
	// IsOver returns true if the rate of failures is over Budget using the time provided.
	IsOver(now time.Time) bool
	// Failure records a number of failures for the time provided.
	Failure(now time.Time, hits int)
	// Attempt records a number of attempts for the time provided.
	Attempt(now time.Time, hits int)
	// Success records a number of successes for the time provided
	Success(now time.Time, hits int)
}

type BudgetSimple struct {
	mutex    sync.Mutex
	interval time.Duration
	last     time.Time
	failures int
	limit    int
}

// NewSimpleBudget creates a new Budget with the specified target failure rate.
// The returned Budget is thread-safe and can be used as a global Budget
// for limiting the total number of retries to a resource from an application,
// regardless of concurrent threads accessing the resource.
//
// 'limit' is the number of failures allowed within the provided interval.
// 'interval' is the length in time the limit is calculated. Once this window is elapsed, the
//  budget is reset.
//
// NewSimpleBudget(50, time.Minute) creates a Budget such that when 50 of the requests
// fail within the one-minute window the Budget is exceeded and IsOver() returns true. IsOver()
// will return false after time has elapsed the 'interval' provided as calculated from the first call
// to Failure() within the interval.
//
// The following illustrates 3 minutes of elapsed time and when IsOver() returns true.
//
// |        12:01:00        |       12:02:00        |        12:03:00        |
// |:----------------------:|:---------------------:|:----------------------:|
// |       0 Failures       |      51 Failures      |       0 Failures       |
// | IsOver() returns False | IsOver() returns True | IsOver() returns False |
//
// Since there were 51 failures during the 12:01:02 interval, IsOver() returns true.
// Once the 12:01:03 interval starts the failure count is reset and IsOver() returns false.
func NewSimpleBudget(limit int, interval time.Duration) Budget {
	return &BudgetSimple{
		limit:    limit,
		interval: interval,
	}
}

// Failure records a number of failures for the given time.
// This method is thread-safe.
func (b *BudgetSimple) Failure(now time.Time, hits int) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	// Shift the window if needed
	b.shiftWindow(now)

	if now.Before(b.last) {
		b.failures += hits
	}
}

// Attempt records a number of attempts for the given time.
// This method is thread-safe.
func (b *BudgetSimple) Attempt(now time.Time, hits int) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	// Shift the window if needed
	b.shiftWindow(now)
}

// Success records a number of successful attempts for the given time.
// This method is thread-safe.
func (b *BudgetSimple) Success(now time.Time, hits int) {
	b.Attempt(now, hits)
}

// IsOver determines if the current failure rate is over the Budget.
// This method is thread-safe.
func (b *BudgetSimple) IsOver(now time.Time) bool {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	// Shift the window if needed
	b.shiftWindow(now)

	return b.failures > b.limit
}

func (b *BudgetSimple) shiftWindow(now time.Time) {
	// Ignore if the current time if before our last shift
	if now.After(b.last) {
		b.failures = 0
		b.last = now.Add(b.interval)
	}
}

// noOpBudget is a Budget implementation that always allows retries.
// It can be used when no Budget control is desired.
type noOpBudget struct{}

// IsOver always returns false for noOpBudget, indicating that the Budget is never exceeded.
func (noOpBudget) IsOver(now time.Time) bool {
	return false
}

// Failure is a no-op for noOpBudget.
func (noOpBudget) Failure(now time.Time, hits int) {}

// Attempt is a no-op for noOpBudget.
func (noOpBudget) Attempt(now time.Time, hits int) {}

// Success is a no-op for noOpBudget.
func (noOpBudget) Success(now time.Time, hits int) {}
