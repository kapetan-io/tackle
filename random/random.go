package random

import (
	"math/rand"
	"strings"
	"time"
)

const NumericRunes = "0123456789"
const LowerAlphaRunes = "abcdefghijklmnopqrstuvwxyz"
const UpperAlphaRunes = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
const AlphaRunes = UpperAlphaRunes + LowerAlphaRunes

// Alpha returns a random string of alpha characters
func Alpha(prefix string, length int) string {
	return Runes(prefix, length, AlphaRunes)
}

// String returns a random string of alpha and numeric characters
func String(prefix string, length int) string {
	return Runes(prefix, length, AlphaRunes, NumericRunes)
}

// One returns one of the strings randomly
func One(items ...string) string {
	return items[rand.Intn(len(items))]
}

// Runes returns a random string made up of characters passed
func Runes(prefix string, length int, runes ...string) string {
	chars := strings.Join(runes, "")
	var bytes = make([]byte, length)

	for i := range bytes {
		bytes[i] = chars[rand.Intn(len(chars))]
	}
	return prefix + string(bytes)
}

// Duration returns a random duration not exceeding the max duration provided
func Duration(min time.Duration, max time.Duration) time.Duration {
	n := rand.Intn(int(max.Nanoseconds()))
	if time.Duration(n) < min {
		return min
	}
	return time.Duration(n)
}

// Slice return a random item from the provided slice
func Slice[S ~[]E, E any](s S) E {
	n := rand.Intn(len(s))
	return s[n]
}
