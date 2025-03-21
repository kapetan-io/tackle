/*
Copyright 2023 Derrick J Wippler

Licensed under the MIT License, you may obtain a copy of the License at

https://opensource.org/license/mit/ or in the root of this code repo

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package retry

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"
)

var ErrCancelRetry = errors.New("retry cancelled by request")

type Interval interface {
	Next(attempts int) time.Duration
}

// IntervalBackOff implements backoff algorithm with a random jitter
//
//	interval := retry.IntervalBackOff{
//		Rand: rand.New(rand.NewSource(time.Now().UnixNano())),
//		Min:    500 * time.Millisecond,
//		Max:    60 * time.Second,
//		Jitter: 0.2, // 20 percent
//		Factor: 1.5,
//	}
type IntervalBackOff struct {
	// Min is the minimum duration of a sleep
	Min time.Duration
	// Max is the maximum duration of a sleep, the exponential calculation will never exceed this duration.
	Max time.Duration
	// Factor is the power by which the minimum duration is applied.
	// NOTE: Use a value of 1.0 or higher, using a factor
	// of 1 or less will not result in an exponential back off
	Factor float64
	// Jitter is the percentage of the Min duration which is used to determine the range of variation when choosing
	// a sleep value. For example: an exponential back off calculation of 1 second with a Jitter of 0.50 (50%)
	// will choose a random sleep duration between 0.5 and 1.5 seconds (500ms, which is 50% of 1 second)
	//
	// The purpose of Jitter is to ensure many client do not all retry at the same time creating additional load
	// on the recovering or downed remote node.
	Jitter float64
	// Rand is the rand instance used to calculate the jitter. If Rand is nil, no jitter is applied.
	Rand *rand.Rand
}

// TODO: Include an example backoff retry interval chart in the documentation, similar too
// https://cloud.google.com/java/docs/reference/google-http-client/1.43.0/com.google.api.client.util.ExponentialBackOff
//

func (b IntervalBackOff) Next(attempts int) time.Duration {
	d := time.Duration(float64(b.Min) * math.Pow(b.Factor, float64(attempts)))
	if b.Rand != nil {
		upper := float64(d) + (float64(d) * b.Jitter)
		lower := float64(d) - (float64(d) * b.Jitter)
		d = time.Duration(lower + b.Rand.Float64()*(upper-lower))
	}
	if d > b.Max {
		return b.Max
	}
	if d < b.Min {
		return b.Min
	}
	return d
}

// BackOffExplain explains the calculations involved in a back off attempt which
// is helpful when deciding upon values for retry.IntervalBackOff. Returned by
// IntervalBackOff.Explain()
type BackOffExplain struct {
	// The minimum range used to calculate jitter
	RangeMin time.Duration
	// The maximum range used to calculate jitter
	RangeMax time.Duration
	// The back off as a calculation of the minimum interval and the PowerOf
	BackOff time.Duration
	// The power of calculation of attempts and factor
	PowerOf float64
	// The backoff with jitter applied
	WithJitter time.Duration
	// The current attempt used in this explanation
	Attempt int
}

// Explain explains the calculation involved based on the number of attempts provided
func (b IntervalBackOff) Explain(attempt int) BackOffExplain {
	// Calc the power of the factor based on attempts
	e := BackOffExplain{Attempt: attempt, PowerOf: math.Pow(b.Factor, float64(attempt))}
	// Backoff is the minimum multiplied by the power
	e.BackOff = time.Duration(float64(b.Min) * e.PowerOf)

	// If we asked for jitter
	if b.Rand != nil {
		percent := float64(e.BackOff) * b.Jitter
		e.RangeMin = time.Duration(float64(e.BackOff) - percent)
		e.RangeMax = time.Duration(float64(e.BackOff) + percent)
		e.WithJitter = time.Duration(float64(e.RangeMin) + b.Rand.Float64()*float64(e.RangeMax-e.RangeMin))
	}
	return e
}

// ExplainString is the same as Explain() but returns the explanation as a string
func (b IntervalBackOff) ExplainString(attempts int) string {
	e := b.Explain(attempts)
	return fmt.Sprintf("Attempt: %d BackOff: %s WithJitter: %s Jitter Range: [%s - %s]",
		e.Attempt, e.BackOff, e.WithJitter, e.RangeMin, e.RangeMax)
}

// IntervalSleep is a constant sleep interval which sleeps for the duration provided before retrying.
type IntervalSleep time.Duration

func (s IntervalSleep) Next(_ int) time.Duration {
	return time.Duration(s)
}

// Policy is the policy retry uses to decide how often how many times an operation should be retried
//
//  policy = retry.Policy{
//  Interval: retry.IntervalBackOff{
//		Rand: rand.New(rand.NewSource(time.Now().UnixNano())),
//		// These values taken from Google Java Client
//		Min:    500 * time.Millisecond,
//		Max:    5 * time.Second,
//		Jitter: 0.2,
//		Factor: 0.5,
//	},
//	Budget:   nil,
//	Attempts: 0,
//}
type Policy struct {
	// Interval is an interface which dictates how long the retry should sleep between attempts. Retry comes with
	// two implementations called retry.IntervalBackOff which implements a backoff and retry.IntervalSleep which
	// is a constant sleep interval with no backoff.
	Interval Interval

	// Budget is the Budget used to determine if a retry should proceed. Budgets block
	// retries until requests are under Budget or the provided context is cancelled.
	// Using a Budget avoids generating excess load on the resource being retried,
	// once it has recovered. It also improves recovery time once the resource
	// has recovered. Set to `nil` to ignore budgets
	// See https://medium.com/yandex/good-retry-bad-retry-an-incident-story-648072d3cee6
	Budget Budget

	// Attempts is the number of "attempts" before an individual retry returns an error to the caller
	// and includes the first attempt, it is a count of the number of "total attempts" that
	// will be attempted.
	Attempts int // 0 for infinite
}

// PolicyDefault is the policy shared by package level Until(), and UntilAttempts() functions
// These values taken from Google Java Client
// https://cloud.google.com/java/docs/reference/google-http-client/1.43.0/com.google.api.client.util.ExponentialBackOff
var PolicyDefault = Policy{
	Interval: IntervalBackOff{
		Rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
		Min:    500 * time.Millisecond,
		Max:    time.Minute,
		Factor: 1.5,
		Jitter: 0.5,
	},
	Budget:   nil,
	Attempts: 0, // Infinite retries
}

// Until retries the provided operation using exponential backoff and the default Budget until the
// context is cancelled
func Until(ctx context.Context, op func(context.Context, int) error) error {
	return Do(ctx, PolicyDefault, op)
}

// UntilAttempts retries the provided operation using exponential backoff and the default Budget until the
// number of attempts has been reached or context is cancelled
func UntilAttempts(ctx context.Context, attempts int, sleep time.Duration, op func(context.Context, int) error) error {
	return Do(ctx, Policy{
		Interval: IntervalBackOff{
			Rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
			Max:    sleep * 10,
			Min:    sleep,
			Jitter: 0.2,
			Factor: 0.5,
		},
		Budget:   PolicyDefault.Budget,
		Attempts: attempts,
	}, op)
}

// Do execute the provided function, retrying using the provided policy.
func Do(ctx context.Context, p Policy, op func(context.Context, int) error) error {
	attempt := 1
	if p.Interval == nil {
		p.Interval = IntervalSleep(time.Second)
	}

	if p.Budget == nil {
		p.Budget = noOpBudget{}
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if p.Budget.IsOver(time.Now()) {
				time.Sleep(p.Interval.Next(attempt))
				p.Budget.Attempt(time.Now(), 1)
				//attempt++
				continue
			}

			err := op(ctx, attempt)
			if err == nil {
				p.Budget.Attempt(time.Now(), 1)
				return err
			}
			p.Budget.Failure(time.Now(), 1)

			// If op function indicates we should cancel the retry
			if errors.Is(err, ErrCancelRetry) {
				// TODO: Test this case
				return ErrCancelRetry
			}

			// If we have exceeded our attempt count
			if p.Attempts != 0 && attempt >= p.Attempts {
				return err
			}

			time.Sleep(p.Interval.Next(attempt))
			attempt++
		}
	}
}
