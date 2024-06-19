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

	for i, _ := range bytes {
		bytes[i] = chars[rand.Intn(len(runes))]
	}
	return prefix + string(bytes)
}

// Duration returns a random duration not exceeding the max duration provided
func Duration(maxDuration time.Duration) time.Duration {
	n := rand.Intn(int(maxDuration.Nanoseconds()))
	if n == 0 {
		return time.Duration(1)
	}
	return time.Duration(n)
}
