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

package retry_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/kapetan-io/tackle/retry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math"
	"math/rand"
	"sync"
	"testing"
	"time"
)

type DoThingRequest struct{}
type DoThingResponse struct{}

type Client struct {
	Err      error
	Attempts int
}

func (c *Client) DoThing(ctx context.Context, r *DoThingRequest, resp *DoThingResponse) error {
	if c.Attempts == 0 {
		return nil
	}
	c.Attempts--
	return c.Err
}

func NewClient() *Client {
	return &Client{}
}

func TestRetry(t *testing.T) {
	c := NewClient()
	var resp DoThingResponse

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// --- Retryable errors are ---
	// 454 - Retry Request
	// 429 - Too Many Requests (Will calculate the appropriate interval to retry based on the response)
	// 500 - Internal Error (Hopefully this is a temporary error, and the server will recover)
	// 502,503,504 - Infrastructure errors which hopefully will resolve on retry.

	c.Err = errors.New("error")
	c.Attempts = 10
	var count int

	t.Run("UntilAttempts", func(t *testing.T) {
		err := retry.UntilAttempts(ctx, 2, time.Second, func(ctx context.Context, attempt int) error {
			err := c.DoThing(ctx, &DoThingRequest{}, &resp)
			if err != nil {
				count++
				return err
			}
			return nil
		})
		require.Error(t, err)
		require.Equal(t, 2, count)
	})

	t.Run("Until", func(t *testing.T) {
		c.Attempts = 5
		count = 0

		_ = retry.Until(ctx, func(ctx context.Context, attempt int) error {
			err := c.DoThing(ctx, &DoThingRequest{}, &resp)
			if err != nil {
				count++
				return err
			}
			return nil
		})
		require.Equal(t, 5, count)
	})

	t.Run("CustomBackoff", func(t *testing.T) {
		customPolicy := retry.Policy{
			Interval: retry.IntervalBackOff{
				Min:    time.Millisecond,
				Max:    time.Millisecond * 100,
				Factor: 2,
			},
			Attempts: 5,
		}

		c.Err = &testError{code: 500}
		c.Attempts = 10
		count = 0

		// Users can define a custom retry policy to suit their needs
		err := retry.Do(ctx, customPolicy, func(ctx context.Context, attempt int) error {
			err := c.DoThing(ctx, &DoThingRequest{}, &resp)
			if err != nil {
				count++
				return err
			}
			return nil
		})

		require.Error(t, err)
		require.Equal(t, 5, count)
	})

	t.Run("ContextCancel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		customPolicy := retry.Policy{
			// No Backoff, just sleep in-between retries
			Interval: retry.IntervalSleep(100 * time.Millisecond),
			// Attempts of 0 indicate infinite retries
			Attempts: 0,
		}

		c.Err = errors.New("error")
		c.Attempts = math.MaxInt
		count = 0

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			err := retry.Do(ctx, customPolicy, func(ctx context.Context, attempt int) error {
				return c.DoThing(ctx, &DoThingRequest{}, &resp)
			})
			require.Error(t, err)
			assert.Equal(t, context.Canceled, err)
			wg.Done()
		}()
		// Cancelling
		time.Sleep(2 * time.Second)
		cancel()
		wg.Wait()
	})

	t.Run("ErrCancelRetry", func(t *testing.T) {
		customPolicy := retry.Policy{
			// No Backoff, just sleep in-between retries
			Interval: retry.IntervalSleep(100 * time.Millisecond),
			// Attempts of 0 indicate infinite retries
			Attempts: 0,
		}

		var retries int
		err := retry.Do(ctx, customPolicy, func(ctx context.Context, attempt int) error {
			if attempt < 5 {
				retries++
				return errors.New("simulate error")
			}
			return fmt.Errorf("manual cancel retry: %w", retry.ErrCancelRetry)
		})
		require.Error(t, err)
		assert.True(t, errors.Is(err, retry.ErrCancelRetry))
		assert.Equal(t, 4, retries)
		assert.Equal(t, "manual cancel retry: retry cancelled by request", err.Error())
	})
}

type testError struct {
	code int
}

func (t testError) Details() map[string]string { return nil }
func (t testError) Error() string              { return "" }
func (t testError) Message() string            { return "" }
func (t testError) Code() int {
	return t.code
}

func TestIntervalBackOff(t *testing.T) {
	p := retry.IntervalBackOff{
		Min:    500 * time.Millisecond,
		Max:    60 * time.Second,
		Factor: 1.5,
	}

	assert.Equal(t, 0.500, p.Next(0).Seconds())
	assert.Equal(t, 0.750, p.Next(1).Seconds())
	assert.Equal(t, 1.125, p.Next(2).Seconds())
	assert.Equal(t, 1.6875, p.Next(3).Seconds())
	assert.Equal(t, 2.53125, p.Next(4).Seconds())
	assert.Equal(t, 3.796875, p.Next(5).Seconds())
	assert.Equal(t, 5.6953125, p.Next(6).Seconds())
	assert.Equal(t, 8.54296875, p.Next(7).Seconds())
	assert.Equal(t, 12.814453125, p.Next(8).Seconds())
	assert.Equal(t, 19.221679687, p.Next(9).Seconds())
	assert.Equal(t, 28.832519531, p.Next(10).Seconds())
	assert.Equal(t, 43.248779296, p.Next(11).Seconds())
	assert.Equal(t, 60.0, p.Next(12).Seconds())
}

func TestIntervalBackOffWithJitter(t *testing.T) {
	p := retry.IntervalBackOff{
		Rand:   rand.New(rand.NewSource(0)),
		Min:    500 * time.Millisecond,
		Max:    60 * time.Second,
		Factor: 1.5,
		Jitter: 0.5,
	}

	assert.Equal(t, 0.722598074, p.Next(0).Seconds())
	assert.Equal(t, 0.558723813, p.Next(1).Seconds())
	assert.Equal(t, 1.300450798, p.Next(2).Seconds())
	assert.Equal(t, 0.935455229, p.Next(3).Seconds())
	assert.Equal(t, 2.196080116, p.Next(4).Seconds())
}

// This test ensures the Explain() calculation agrees with the Next()
// calculation.
func TestExplainAgrees(t *testing.T) {
	t.Run("backoff", func(t *testing.T) {
		i := retry.IntervalBackOff{
			Min:    500 * time.Millisecond,
			Max:    60 * time.Second,
			Factor: 1.5,
		}
		e := retry.IntervalBackOff{
			Min:    500 * time.Millisecond,
			Max:    60 * time.Second,
			Factor: 1.5,
		}

		assert.Equal(t, i.Next(0), e.Explain(0).BackOff)
		assert.Equal(t, i.Next(1), e.Explain(1).BackOff)
		assert.Equal(t, i.Next(2), e.Explain(2).BackOff)
		assert.Equal(t, i.Next(3), e.Explain(3).BackOff)
	})

	t.Run("with-jitter", func(t *testing.T) {
		i := retry.IntervalBackOff{
			Rand:   rand.New(rand.NewSource(0)),
			Min:    500 * time.Millisecond,
			Max:    60 * time.Second,
			Factor: 1.5,
			Jitter: 0.5,
		}
		e := retry.IntervalBackOff{
			Rand:   rand.New(rand.NewSource(0)),
			Min:    500 * time.Millisecond,
			Max:    60 * time.Second,
			Factor: 1.5,
			Jitter: 0.5,
		}

		assert.Equal(t, i.Next(0), e.Explain(0).WithJitter)
		assert.Equal(t, i.Next(1), e.Explain(1).WithJitter)
		assert.Equal(t, i.Next(2), e.Explain(2).WithJitter)
		assert.Equal(t, i.Next(3), e.Explain(3).WithJitter)
	})
}

func TestExplainString(t *testing.T) {
	p := retry.IntervalBackOff{
		Rand:   rand.New(rand.NewSource(0)),
		Min:    500 * time.Millisecond,
		Max:    60 * time.Second,
		Factor: 1.5,
		Jitter: 0.5,
	}

	t.Logf("retry.IntervalBackOff{\n\tMin: %v\n\t"+
		"Max: %v\n\tJitter: %v\n\tFactor: %v\n}\n", p.Min, p.Max, p.Jitter, p.Factor)

	for attempts := 0; attempts < 10; attempts++ {
		t.Logf("%s\n", p.ExplainString(attempts))
	}
}
