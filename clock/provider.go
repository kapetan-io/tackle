package clock

import "time"

// NewProvider creates a new instance of a clock provider that can be independently frozen, advanced, and tracked.
// This allows some packages or instances to manipulate time without affecting the global clock, enabling other
// packages with separate provider instances to maintain their own distinct timelines.
func NewProvider() *Provider {
	return &Provider{}
}

// Freeze "freezes" time
func (cp *Provider) Freeze(now time.Time) {
	cp.setProvider(&frozenTime{frozenAt: now, now: now})
}

// UnFreeze "unfreezes" time
func (cp *Provider) UnFreeze() {
	cp.setProvider(realtime)
}

// Advance makes the deterministic time move forward by the specified duration, firing timers along the way in the
// natural order. It returns how much time has passed since it was frozen. So you can assert on the return value
// in tests to make it explicit where you stand on the deterministic timescale.
func (cp *Provider) Advance(d time.Duration) time.Duration {
	ft, ok := cp.getProvider().(*frozenTime)
	if !ok {
		panic("Freeze time first!")
	}
	ft.advance(d)
	return Now().Sub(ft.frozenAt)
}

// Wait4Scheduled blocks until either there are n or more scheduled events, or the timeout elapses. It returns true
// if the wait condition has been met before the timeout expired, false otherwise.
func (cp *Provider) Wait4Scheduled(count int, timeout time.Duration) bool {
	return cp.getProvider().Wait4Scheduled(count, timeout)
}

// NewTimer see time.NewTimer.
func (cp *Provider) NewTimer(d time.Duration) Timer {
	return cp.getProvider().NewTimer(d)
}

// After see time.After.
func (cp *Provider) After(d time.Duration) <-chan time.Time {
	return cp.getProvider().After(d)
}

// NewTicker see time.Ticker.
func (cp *Provider) NewTicker(d time.Duration) Ticker {
	return cp.getProvider().NewTicker(d)
}

// Tick see time.Tick.
func (cp *Provider) Tick(d time.Duration) <-chan time.Time {
	return cp.getProvider().Tick(d)
}

// Now see time.Now.
func (cp *Provider) Now() time.Time {
	return cp.getProvider().Now()
}

// Sleep see time.Sleep.
func (cp *Provider) Sleep(d time.Duration) {
	cp.getProvider().Sleep(d)
}
