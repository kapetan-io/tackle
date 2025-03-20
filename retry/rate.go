package retry

import (
	"fmt"
	"math"
	"time"
)

// Rate calculates the hit rate per second within a requested window size.
// It uses a simplistic version of a ring buffer to track hits within buckets.
type Rate struct {
	// last is the last time Add or Rate was called. It is used to
	// identify the last bucket in buckets hits were added.
	last time.Time

	// buckets is the time buckets where each bucket represents one second.
	// and the number in the bucket represents how many hits occurred during
	// that one second interval. buckets is a simple ring buffer.
	buckets []int

	// windowSize is the size of the window, which is smaller than the actual
	// size of the buckets.
	windowSize int

	// pos is the current position inside the buckets, which is treated
	// like a ring buffer.
	pos int
}

func NewRate(windowSize int) *Rate {
	return &Rate{
		buckets:    make([]int, windowSize+1),
		windowSize: windowSize,
	}
}

func (m *Rate) Add(now time.Time, hits int) {
	if now.Before(m.last) {
		return
	}

	m.shiftWindow(now)
	m.buckets[m.pos] += hits
}

func (m *Rate) Rate(now time.Time) float64 {
	if now.Before(m.last) {
		return math.NaN()
	}

	m.shiftWindow(now)

	var first, sum float64
	var bucketsUsed int
	pos := m.pos

	for i := 0; i < len(m.buckets); i++ {
		pos = (pos + 1) % len(m.buckets)
		if m.buckets[pos] == 0 {
			continue
		}
		bucketsUsed++

		if first == 0 {
			first = float64(m.buckets[pos])
			continue
		}
		sum += float64(m.buckets[pos])
	}
	var seconds time.Duration

	// Avoid adding weight to a window that isn't full
	if bucketsUsed < len(m.buckets) {
		seconds = time.Duration(bucketsUsed-1) * time.Second
		seconds += m.last.Sub(roundDown(m.last))
		sum += first
	} else {
		seconds = time.Duration(m.windowSize) * time.Second
		weight := 1.0 - float64(m.last.Sub(roundDown(m.last)))/float64(time.Second)
		sum += weight * first
	}

	if seconds < time.Second {
		seconds = time.Second
	}

	result := sum / seconds.Seconds()
	return result
}

// shiftWindow manages moving the window according to the time provided.
// Although `windowSize` is the size of the window, we keep one additional bucket
// `windowSize+1` so we can preform a weighted average using the older buckets
// outside our window.
func (m *Rate) shiftWindow(now time.Time) {
	defer func() {
		m.last = now
	}()

	rt := roundDown(now)
	// If this is our first time, or the current time precedes or is equal to our
	// last update, no window change is needed as time has not advanced.
	if m.last.IsZero() || !rt.After(m.last) {
		return
	}

	// Calculate the number of buckets to advance
	adv := int(rt.Sub(roundDown(m.last)) / time.Second)
	if adv <= 0 {
		panic(fmt.Sprintf("assert failed: adv = %d; rt = %v, m.last = %v", adv, rt, m.last))
	}

	// Avoid advancing further than the size of our ring buffer
	if adv > len(m.buckets) {
		adv = len(m.buckets)
	}

	// advance through the buckets starting at head and
	// clear any hits for each bucket we advance.
	pos := m.pos
	for i := 0; i < adv; i++ {
		pos = (pos + 1) % len(m.buckets)
		m.buckets[pos] = 0
	}
	m.pos = (m.pos + adv) % len(m.buckets)
	m.last = m.last.Add(time.Duration(adv) * time.Second)
}

// roundDown rounds the current time down to the nearest second
func roundDown(now time.Time) time.Time {
	r := now.Round(time.Second)
	if r.After(now) {
		r = r.Add(-time.Second)
	}
	return r
}
