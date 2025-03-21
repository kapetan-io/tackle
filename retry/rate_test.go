package retry

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestRate(t *testing.T) {
	t.Run("TestCases", func(t *testing.T) {

		cases := []struct {
			name   string
			calls  []int
			expect string
		}{
			{
				name:   "one-second",
				calls:  []int{1},
				expect: "0.10",
			},
			{
				name:   "two-seconds",
				calls:  []int{1, 1},
				expect: "0.20",
			},
			{
				name:   "three-seconds",
				calls:  []int{1, 1, 1},
				expect: "0.30",
			},
			{
				name:   "ten-seconds",
				calls:  []int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
				expect: "1.00",
			},
			{
				name: "avg",
				calls: []int{
					5, // outside the window
					5, 5, 5, 5, 5, 5, 5, 5, 5, 1},
				expect: "4.60",
			},
			{
				name: "avg-large",
				calls: []int{
					10000, 2, 2, 2, 2, 2, 2, 2, 2, 2,
				},
				expect: "1001.80",
			},
			{
				name: "shift-window",
				calls: []int{
					2, 2, 2, 2, // removed by window shift
					5, // outside the window
					1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
				},
				expect: "1.00",
			},
		}

		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				mr := NewRate(time.Second, 10)
				tm := time.Date(2018, time.February, 22, 22, 24, 53, 200000000, time.UTC)
				for _, n := range c.calls {
					tm = tm.Add(time.Second)
					// Call Add() 'n' times
					for j := 0; j < n; j++ {
						mr.Add(tm, 1)
					}
				}
				assert.Equal(t, c.expect, fmt.Sprintf("%.2f", mr.Rate(tm)))
			})
		}
	})

	t.Run("TimeGap", func(t *testing.T) {
		mr := NewRate(time.Second, 10)
		now := time.Date(2018, time.February, 22, 22, 24, 53, 200000000, time.UTC)
		mr.Add(now, 5)
		assert.Equal(t, "0.50", fmt.Sprintf("%.2f", mr.Rate(now)))

		now = now.Add(time.Minute)
		mr.Add(now, 5)
		assert.Equal(t, "0.50", fmt.Sprintf("%.2f", mr.Rate(now)))

		now = now.Add(time.Minute)
		assert.Equal(t, "0.00", fmt.Sprintf("%.2f", mr.Rate(now)))

		now = now.Add(time.Minute)
		mr.Add(now, 5)
		assert.Equal(t, "0.50", fmt.Sprintf("%.2f", mr.Rate(now)))
		//t.Logf("AFTER  mr = %+v", mr)
	})

	t.Run("TimeGapExtended", func(t *testing.T) {
		mr := NewRate(time.Second, 10)
		now := time.Date(2018, time.February, 22, 22, 24, 53, 200000000, time.UTC)
		mr.Add(now, 5)
		assert.Equal(t, "0.50", fmt.Sprintf("%.2f", mr.Rate(now)))

		// Elapse some large amount of time outside our window
		now = now.Add(time.Minute * 20)

		// Now fill up the window
		mr.Add(now, 5)
		now = now.Add(time.Second)
		mr.Add(now, 5)
		now = now.Add(time.Second)
		mr.Add(now, 5)
		now = now.Add(time.Second)
		mr.Add(now, 5)
		now = now.Add(time.Second)
		mr.Add(now, 5)
		now = now.Add(time.Second)
		mr.Add(now, 5)
		now = now.Add(time.Second)
		mr.Add(now, 5)
		now = now.Add(time.Second)
		mr.Add(now, 5)
		now = now.Add(time.Second)
		mr.Add(now, 5)
		now = now.Add(time.Second)
		mr.Add(now, 5)
		now = now.Add(time.Second)
		mr.Add(now, 1)
		assert.Equal(t, "4.60", fmt.Sprintf("%.2f", mr.Rate(now)))
	})
}

func BenchmarkRate(b *testing.B) {
	m := NewRate(time.Second, 60)
	now := time.Date(2018, time.February, 22, 22, 24, 53, 200000000, time.UTC)

	b.Run("Moving Rate", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			now = now.Add(time.Second)
			m.Add(now, 5)
		}
		m.Rate(now.Add(time.Second))
		b.ReportAllocs()
	})
}
