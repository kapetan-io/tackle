//go:build !clock_mutex

package clock

var (
	provider Clock = realtime
)

func setProvider(p Clock) {
	provider = p
}

func getProvider() Clock {
	return provider
}
