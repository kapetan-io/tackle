package retry

import (
	"math"
	"time"
)

// Rate calculates the hit rate per second within a requested window size.
// It uses a simplistic version of a ring buffer to track hits within buckets.
type Rate struct {
	// last is the last time Add or Rate was called to avoid adding
	// to buckets in the past
	last time.Time

	// buckets is the time buckets where each bucket represents one interval.
	// and the number in the bucket represents how many hits occurred during
	// that interval. buckets is a simple ring buffer.
	buckets []int

	// interval is the interval each bucket in the sliding window represents
	interval time.Duration

	// pos is the current position inside the ring buffer
	pos int

	// total is the sum of all hits in the current window
	total int
}

func NewRate(interval time.Duration, buckets int) *Rate {
	return &Rate{
		buckets:  make([]int, buckets),
		interval: interval,
	}
}

func (m *Rate) Add(now time.Time, hits int) {
	// Ignore calls with timestamps earlier than the last recorded time
	if now.Before(m.last) {
		return
	}

	// Shift the window if necessary
	m.shiftWindow(now)

	// Add hits to the current bucket
	m.buckets[m.pos] += hits

	// Update the total hits
	m.total += hits
}

func (m *Rate) Rate(now time.Time) float64 {
	// Return NaN for calls with timestamps earlier than the last recorded time
	if now.Before(m.last) {
		return math.NaN()
	}

	// Shift the window if necessary
	m.shiftWindow(now)

	// If there are no hits, return 0
	if m.total == 0 {
		return 0.0
	}

	// Calculate the rate: total hits / window duration in seconds
	windowDuration := time.Duration(len(m.buckets)) * m.interval
	return float64(m.total) / windowDuration.Seconds()
}

// shiftWindow manages moving the window according to the time provided.
func (m *Rate) shiftWindow(now time.Time) {
	defer func() {
		// Update the last recorded time
		m.last = now
	}()

	// Round down the current time to the nearest interval
	rt := roundDown(now)

	// If this is our first time, or the current time precedes or is equal to our
	// last update, no window change is needed as time has not advanced.
	if m.last.IsZero() || !rt.After(m.last) {
		return
	}

	// Calculate the number of buckets to advance
	adv := int(rt.Sub(roundDown(m.last)) / m.interval)
	if adv <= 0 {
		return
	}

	// Avoid advancing further than the size of our ring buffer
	if adv > len(m.buckets) {
		adv = len(m.buckets)
	}

	// Advance through the buckets, clearing hits and updating the total
	for i := 0; i < adv; i++ {
		m.pos = (m.pos + 1) % len(m.buckets)
		m.total -= m.buckets[m.pos]
		m.buckets[m.pos] = 0
	}
}

// roundDown rounds the current time down to the nearest interval
func roundDown(now time.Time) time.Time {
	return now.Truncate(time.Second)
}
