package clock_test

import (
	"github.com/kapetan-io/tackle/clock"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewProvider(t *testing.T) {
	// Create a new provider and freeze time.
	p1 := clock.NewProvider()
	now := clock.Now()
	p1.Freeze(now)

	// Advance the first provider by 10 seconds.
	p1.Advance(10 * clock.Second)
	assert.Equal(t, 10, int(p1.Now().Sub(now).Seconds()))

	// Create a second provider and freeze it.
	p2 := clock.NewProvider()
	p2.Freeze(clock.Now())

	// Advance the second provider by 30 seconds.
	p2.Advance(30 * clock.Second)

	assert.Equal(t, 30, int(p2.Now().Sub(now).Seconds()))

	// Advance the second provider by another 5 seconds.
	p2.Advance(5 * clock.Second)
	assert.Equal(t, 35, int(p2.Now().Sub(now).Seconds()))
}
