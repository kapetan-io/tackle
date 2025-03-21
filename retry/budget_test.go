package retry_test

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"github.com/kapetan-io/tackle/retry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewSimpleBudget(t *testing.T) {
	t.Run("InitialState", func(t *testing.T) {
		budget := retry.NewSimpleBudget(50, time.Minute)
		now := time.Now()
		assert.False(t, budget.IsOver(now))
	})

	t.Run("OverBudget", func(t *testing.T) {
		budget := retry.NewSimpleBudget(50, time.Second)
		now := time.Now()

		// Add failures to exceed the Budget
		budget.Failure(now, 50)

		assert.False(t, budget.IsOver(now))
		budget.Failure(now, 1)
		assert.True(t, budget.IsOver(now))
	})

	t.Run("RecoveryAfterIntervalElapsed", func(t *testing.T) {
		budget := retry.NewSimpleBudget(50, time.Second)
		now := time.Now()

		// Add failures to exceed the Budget
		budget.Failure(now, 51)

		assert.True(t, budget.IsOver(now))

		now = now.Add(time.Second + time.Millisecond)
		assert.False(t, budget.IsOver(now))
	})

	t.Run("IgnoreAttemptAndSuccess", func(t *testing.T) {
		budget := retry.NewSimpleBudget(50, time.Second)
		now := time.Now()

		// Only add attempts && successes
		budget.Attempt(now, 100)
		budget.Success(now, 100)

		assert.False(t, budget.IsOver(now))
	})

	t.Run("TimeDecay", func(t *testing.T) {
		budget := retry.NewSimpleBudget(19, time.Second)
		now := time.Now()

		// Add failures to exceed the Budget
		budget.Failure(now, 20)

		assert.True(t, budget.IsOver(now))

		// Move time forward by 30 seconds
		futureTime := now.Add(30 * time.Second)
		assert.False(t, budget.IsOver(futureTime))

		// Move time forward by another 31 seconds (total 61 seconds)
		futureTime = now.Add(61 * time.Second)
		assert.False(t, budget.IsOver(futureTime))

		// Add a small number of failures
		budget.Failure(futureTime, 5)

		assert.False(t, budget.IsOver(futureTime))

		// Add more failures to exceed the Budget again
		budget.Failure(futureTime, 15)
		assert.True(t, budget.IsOver(futureTime))
	})
}

func TestBudgetWithDo(t *testing.T) {
	ctx := context.Background()

	t.Run("UnderBudget", func(t *testing.T) {
		budget := retry.NewSimpleBudget(50, time.Second)
		policy := retry.Policy{
			Interval: retry.IntervalSleep(100 * time.Millisecond),
			Budget:   budget,
			Attempts: 10,
		}

		var successCount, failureCount, lastAttempt int
		// Should retry 6 times, 5 failures, and one attempt, never exceeding the Budget
		err := retry.Do(ctx, policy, func(ctx context.Context, attempt int) error {
			lastAttempt = attempt
			if attempt <= 5 {
				failureCount++
				return errors.New("simulated failure")
			}
			successCount++
			return nil
		})
		assert.NoError(t, err)
		assert.Equal(t, 5, failureCount)
		assert.Equal(t, 1, successCount)
		assert.Equal(t, 6, lastAttempt)
	})

	t.Run("OverBudget", func(t *testing.T) {
		budget := retry.NewSimpleBudget(50, time.Second) // Set a very low ratio to trigger Budget exceeded
		policy := retry.Policy{
			Interval: retry.IntervalSleep(5 * time.Millisecond),
			Budget:   budget,
			Attempts: 60, // Increase attempts to allow Budget to be exceeded
		}

		var err error
		var successCount, failureCount int
		for i := 0; i < 10; i++ {
			err = retry.Do(ctx, policy, func(ctx context.Context, attempt int) error {
				successCount++
				return nil
			})
			require.NoError(t, err)
		}
		assert.Equal(t, 10, successCount)

		err = retry.Do(ctx, policy, func(ctx context.Context, attempt int) error {
			if attempt <= 51 {
				failureCount++
				return errors.New("simulated failure")
			}
			successCount++
			return nil
		})
		require.NoError(t, err)
		assert.Equal(t, 51, failureCount)
		assert.Equal(t, 11, successCount)
	})
}

type Point struct {
	Time    time.Time
	Success int
	Failed  int
}

func TestBudgetGraph(t *testing.T) {
	t.Skip("used for graphing budgets vs backoff recovery time")
	client := http.Client{
		Transport: &http.Transport{
			ForceAttemptHTTP2:     true,
			MaxIdleConnsPerHost:   100_000,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	report(t, retry.Policy{
		Interval: retry.IntervalBackOff{
			Rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
			Min:    time.Millisecond,
			Max:    500 * time.Millisecond,
			Factor: 1.01,
			Jitter: 0.50,
		},
		Budget:   nil,
		Attempts: 0,
	}, client, "no-budget")

	report(t, retry.Policy{
		Interval: retry.IntervalBackOff{
			Rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
			Min:    time.Millisecond,
			Max:    500 * time.Millisecond,
			Factor: 1.01,
			Jitter: 0.50,
		},
		Budget:   retry.NewSimpleBudget(50, time.Second),
		Attempts: 0,
	}, client, "with-budget")
}

func report(t *testing.T, policy retry.Policy, client http.Client, prefix string) {
	var hits []Point
	var upTime []Point
	var mutex sync.Mutex
	var down atomic.Bool

	// TODO: Remove
	prefix = fmt.Sprintf("/Users/thrawn/Development/marimo/%s", prefix)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if down.Load() {
			mutex.Lock()
			hits = append(hits, Point{Time: time.Now(), Failed: 1})
			mutex.Unlock()

			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("NOT OK"))
			return
		}
		mutex.Lock()
		hits = append(hits, Point{Time: time.Now(), Success: 1})
		mutex.Unlock()
		time.Sleep(time.Millisecond * time.Duration(rand.Intn(10)))
		//time.Sleep(5 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())

	// Wait until we are at the nearest second before starting the test to
	// ensure round up/down doesn't skew results in un-expected ways
	now := time.Now()
	start := roundUp(now, time.Second)
	fmt.Printf("Run Time: %+v\n", now)
	time.Sleep(start.Sub(now))

	start = time.Now()
	fmt.Printf("Start Time: %+v\n", start)
	start = start.Round(time.Second)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				_ = retry.Do(ctx, policy, func(ctx context.Context, i int) error {
					return request(&client, server.URL)
				})
				// Wait to be cancelled or for our next request. This simulates
				// a client who is doing work in between making requests to the server.
				select {
				case <-ctx.Done():
					return
				//case <-time.After(time.Duration(rand.Intn(100)) * time.Millisecond):
				default: // <-- to simulate no time between requests
				}
			}
		}()
	}

	upTime = append(upTime, Point{Time: time.Now().Round(time.Millisecond * 250), Success: 1500})
	time.Sleep(2 * time.Second)
	upTime = append(upTime, Point{Time: time.Now().Round(time.Millisecond * 250), Success: 1500})
	upTime = append(upTime, Point{Time: time.Now().Round(time.Millisecond * 250), Success: 0})
	down.Store(true)
	time.Sleep(4 * time.Second)
	upTime = append(upTime, Point{Time: time.Now().Round(time.Millisecond * 250), Success: 0})
	upTime = append(upTime, Point{Time: time.Now().Round(time.Millisecond * 250), Success: 1500})
	time.Sleep(100 * time.Millisecond)
	down.Store(false)
	time.Sleep(1900 * time.Millisecond)
	upTime = append(upTime, Point{Time: time.Now().Round(time.Millisecond * 250), Success: 1500})

	// Cancel the context and wait for the go routines to end
	cancel()
	wg.Wait()
	stop := time.Now()

	r := rollup(hits)
	writeRollup(t, r, start, fmt.Sprintf("%s-data.csv", prefix))
	writeUpTime(t, upTime, start, fmt.Sprintf("%s-uptime.csv", prefix))
	writeInterval(t, start, stop.Add(250*time.Millisecond), 250*time.Millisecond,
		fmt.Sprintf("%s-intervals.csv", prefix))
}

func writeInterval(t *testing.T, start time.Time, stop time.Time, interval time.Duration, name string) {
	f, err := os.Create(name)
	if err != nil {
		panic(err)
	}
	w := csv.NewWriter(f)
	_ = w.Write([]string{"time"})
	start = start.Round(interval)
	for i := 0; ; i++ {
		n := start.Add(interval * time.Duration(i))
		if n.After(stop) {
			break
		}
		_ = w.Write([]string{fmt.Sprintf("%.1f", n.Sub(start).Seconds())})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		panic(err)
	}
	_ = f.Close()
	t.Logf("Wrote: %s", name)
}

func writeUpTime(t *testing.T, upTime []Point, now time.Time, name string) {
	f, err := os.Create(name)
	if err != nil {
		panic(err)
	}
	w := csv.NewWriter(f)
	_ = w.Write([]string{"time", "up"})

	for _, point := range upTime {
		ts := point.Time.Sub(now).Seconds()
		_ = w.Write([]string{fmt.Sprintf("%.1f", ts), fmt.Sprintf("%d", point.Success)})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		panic(err)
	}
	_ = f.Close()
	t.Logf("Wrote: %s", name)
}

func writeRollup(t *testing.T, rollup []Point, now time.Time, name string) {
	f, err := os.Create(name)
	if err != nil {
		panic(err)
	}
	w := csv.NewWriter(f)
	_ = w.Write([]string{"time", "success", "failed"})

	for _, point := range rollup {
		ts := point.Time.Sub(now).Seconds()
		_ = w.Write([]string{
			fmt.Sprintf("%.1f", ts),
			fmt.Sprintf("%d", point.Success),
			fmt.Sprintf("%d", point.Failed),
		})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		panic(err)
	}
	_ = f.Close()
	t.Logf("Wrote: %s", name)
}

func request(client *http.Client, url string) error {
	resp, err := client.Get(url)
	if err != nil {
		panic(err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read the response body to allow connection reuse
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return errors.New("request failed")
	}
	return nil
}

func rollup(series []Point) []Point {
	buckets := make(map[time.Time]*Point)
	for _, p := range series {
		key := roundUp(p.Time, 100*time.Millisecond)
		if o, ok := buckets[key]; ok {
			o.Failed += p.Failed
			o.Success += p.Success
		} else {
			p.Time = key
			buckets[key] = &Point{Time: key, Success: p.Success, Failed: p.Failed}
		}
	}
	var result []Point
	for k, v := range buckets {
		result = append(result, Point{Time: k, Success: v.Success, Failed: v.Failed})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Time.Before(result[j].Time)
	})
	return result
}

// roundUp rounds the current time up
func roundUp(now time.Time, interval time.Duration) time.Time {
	r := now.Round(interval)
	if r.Before(now) {
		r = r.Add(interval)
	}
	return r
}
