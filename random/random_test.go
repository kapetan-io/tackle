package random_test

import (
	"fmt"
	"github.com/kapetan-io/tackle/random"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestString(t *testing.T) {
	for _, tc := range []struct {
		msg            string
		prefix         string
		length         int
		contains       string
		expectedLength int
	}{{
		msg:            "only made of alpha and numeric characters",
		prefix:         "",
		length:         10,
		contains:       random.AlphaRunes + random.NumericRunes,
		expectedLength: 10,
	}, {
		msg:            "with prefix and add to length",
		prefix:         "abc",
		length:         5,
		contains:       random.AlphaRunes + random.NumericRunes,
		expectedLength: 5 + 3, // = len("abc")
	}} {
		t.Run(tc.msg, func(t *testing.T) {
			res := random.String(tc.prefix, tc.length)
			t.Logf("Random String: %s", res)
			assert.Equal(t, tc.expectedLength, len(res))
			assert.Contains(t, res, tc.prefix)
			for _, ch := range res {
				assert.Contains(t, tc.contains, fmt.Sprintf("%c", ch))
			}
		})
	}
}

func TestAlpha(t *testing.T) {
	for _, tc := range []struct {
		msg            string
		prefix         string
		length         int
		contains       string
		expectedLength int
	}{{
		msg:            "only made of alpha characters",
		prefix:         "",
		length:         10,
		contains:       random.AlphaRunes,
		expectedLength: 10,
	}, {
		msg:            "with prefix and add to length",
		prefix:         "abc",
		length:         5,
		contains:       random.AlphaRunes,
		expectedLength: 5 + 3, // = len("abc")
	}} {
		t.Run(tc.msg, func(t *testing.T) {
			res := random.Alpha(tc.prefix, tc.length)
			t.Logf("Random Alpha: %s", res)
			assert.Equal(t, tc.expectedLength, len(res))
			assert.Contains(t, res, tc.prefix)
			for _, ch := range res {
				assert.Contains(t, tc.contains, fmt.Sprintf("%c", ch))
			}
		})
	}
}

func TestItem(t *testing.T) {
	for _, tc := range []struct {
		msg      string
		items    []string
		expected string
	}{{
		msg:   "one of the given list",
		items: []string{"com", "net", "org"},
	}} {
		t.Run(tc.msg, func(t *testing.T) {
			res := random.One(tc.items...)
			assert.Contains(t, tc.items, res)
		})
	}
}

func TestDuration(t *testing.T) {
	d := random.Duration(time.Second)
	t.Logf("duration: %s", d)
	assert.True(t, d < time.Second)
	assert.True(t, d.Nanoseconds() != 0)

	d = random.Duration(time.Minute)
	t.Logf("duration: %s", d)
	assert.True(t, d < time.Minute)
	assert.True(t, d.Nanoseconds() != 0)

	d = random.Duration(60 * time.Minute)
	t.Logf("duration: %s", d)
	assert.True(t, d < 60*time.Minute)
	assert.True(t, d.Nanoseconds() != 0)
}
