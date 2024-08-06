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

type Provider struct {
	provider Clock
}

func (cp *Provider) setProvider(c Clock) {
	cp.provider = c
}

func (cp *Provider) getProvider() Clock {
	return cp.provider
}
