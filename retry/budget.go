package retry

import (
	"fmt"
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
}

type budget struct {
	mutex   sync.Mutex
	ratio   float64
	attempt *Rate
	failure *Rate
}

// NewBudget creates a new Budget with the specified target failure rate.
// The returned budget is thread-safe and can be used as a global budget
// for limiting the total number of retries to a resource from an application,
// regardless of concurrent threads accessing the resource.
//
// 'ratio' is the maximum ratio of failures to successes allowed within a 60 second window.
func NewBudget(ratio float64) Budget {
	return &budget{
		attempt: NewRate(60), // 1-minute window
		failure: NewRate(60), // 1-minute window
		ratio:   ratio,
	}
}

// Failure records a number of failures for the given time.
// This method is thread-safe.
func (b *budget) Failure(now time.Time, hits int) {
	defer b.mutex.Unlock()
	b.mutex.Lock()
	b.failure.Add(now, hits)
	b.attempt.Add(now, 0)
}

// Attempt records a number of attempts for the given time.
// This method is thread-safe.
func (b *budget) Attempt(now time.Time, hits int) {
	defer b.mutex.Unlock()
	b.mutex.Lock()
	b.failure.Add(now, 0)
	b.attempt.Add(now, hits)
}

// IsOver determines if the current failure rate is over the budget.
// This method is thread-safe.
func (b *budget) IsOver(now time.Time) bool {
	defer b.mutex.Unlock()
	b.mutex.Lock()

	failureRate := b.failure.Rate(now)
	attemptRate := b.attempt.Rate(now)

	// If there are no failures, we're not over budget
	if failureRate == 0 {
		return false
	}

	if attemptRate == 0 {
		return true
	}

	fmt.Printf("Failure rate: %f\n", failureRate)
	fmt.Printf("Attempt rate: %f\n", attemptRate)
	// We're over budget if the ratio of failures to successes exceeds the specified ratio
	fmt.Printf("Failure ratio: %f\n", failureRate/attemptRate)
	if failureRate < attemptRate {

	}
	return failureRate/attemptRate > b.ratio
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
