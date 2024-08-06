//go:build clock_mutex

package clock

import "sync"

var (
	providerMu sync.RWMutex
	provider   Clock = realtime
)

func setProvider(p Clock) {
	providerMu.Lock()
	provider = p
	providerMu.Unlock()
}

func getProvider() Clock {
	providerMu.RLock()
	p := provider
	providerMu.RUnlock()
	return p
}

type Provider struct {
	mutex    sync.RWMutex
	provider Clock
}

func (p *Provider) setProvider(c Clock) {
	defer p.mutex.Unlock()
	p.mutex.Lock()
	p.provider = c
}

func (p *Provider) getProvider() Clock {
	defer p.mutex.RUnlock()
	p.mutex.RLock()
	return p.provider
}
