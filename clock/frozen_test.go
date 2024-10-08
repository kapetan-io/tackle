package clock_test

import (
	"fmt"
	"github.com/kapetan-io/tackle/clock"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

func TestFreezeUnFreeze(t *testing.T) {
	clock.Freeze(clock.Now()).UnFreeze()
}

type FrozenSuite struct {
	suite.Suite
	epoch time.Time
}

func TestFrozenSuite(t *testing.T) {
	suite.Run(t, new(FrozenSuite))
}

func (s *FrozenSuite) SetupSuite() {
	var err error
	s.epoch, err = time.Parse(time.RFC3339, "2009-02-19T00:00:00Z")
	s.Require().NoError(err)
}

func (s *FrozenSuite) SetupTest() {
	clock.Freeze(s.epoch)
}

func (s *FrozenSuite) TearDownTest() {
	clock.UnFreeze()
}

func (s *FrozenSuite) TestAdvanceNow() {
	s.Require().Equal(s.epoch, clock.Now())
	s.Require().Equal(42*time.Millisecond, clock.Advance(42*time.Millisecond))
	s.Require().Equal(s.epoch.Add(42*time.Millisecond), clock.Now())
	s.Require().Equal(55*time.Millisecond, clock.Advance(13*time.Millisecond))
	s.Require().Equal(74*time.Millisecond, clock.Advance(19*time.Millisecond))
	s.Require().Equal(s.epoch.Add(74*time.Millisecond), clock.Now())
}

func (s *FrozenSuite) TestSleep() {
	hits := make(chan int, 100)

	delays := []int{60, 100, 90, 131, 999, 5}
	for i, tc := range []struct {
		Name string
		fn   func(delayMs int)
	}{{
		Name: "Sleep",
		fn: func(delay int) {
			clock.Sleep(time.Duration(delay) * time.Millisecond)
			hits <- delay
		},
	}, {
		Name: "After",
		fn: func(delay int) {
			<-clock.After(time.Duration(delay) * time.Millisecond)
			hits <- delay
		},
	}, {
		Name: "AfterFunc",
		fn: func(delay int) {
			clock.AfterFunc(time.Duration(delay)*time.Millisecond,
				func() {
					hits <- delay
				})
		},
	}, {
		Name: "NewTimer",
		fn: func(delay int) {
			t := clock.NewTimer(time.Duration(delay) * time.Millisecond)
			<-t.C()
			hits <- delay
		},
	}} {
		fmt.Printf("Test case #%d: %s", i, tc.Name)
		for _, delay := range delays {
			go func(delay int) {
				tc.fn(delay)
			}(delay)
		}

		// Wait for all go routines to register their scheduled timers
		clock.Wait4Scheduled(len(delays), time.Second)

		runningMs := 0
		for i, delayMs := range []int{5, 60, 90, 100, 131, 999} {
			fmt.Printf("Checking timer #%d, delay=%d\n", i, delayMs)
			delta := delayMs - runningMs - 1
			clock.Advance(time.Duration(delta) * time.Millisecond)
			// Check before each timer deadline that it is not triggered yet.
			s.assertHits(hits, []int{})

			// When
			clock.Advance(1 * time.Millisecond)

			// Then
			s.assertHits(hits, []int{delayMs})

			runningMs += delta + 1
		}

		clock.Advance(1000 * time.Millisecond)
		s.assertHits(hits, []int{})
	}
}

// Timers scheduled to trigger at the same time do that in the order they were
// created.
func (s *FrozenSuite) TestSameTime() {
	var hits []int

	clock.AfterFunc(100, func() { hits = append(hits, 3) })
	clock.AfterFunc(100, func() { hits = append(hits, 1) })
	clock.AfterFunc(99, func() { hits = append(hits, 2) })
	clock.AfterFunc(100, func() { hits = append(hits, 5) })
	clock.AfterFunc(101, func() { hits = append(hits, 4) })
	clock.AfterFunc(101, func() { hits = append(hits, 6) })

	// When
	clock.Advance(100)

	// Then
	s.Require().Equal([]int{2, 3, 1, 5}, hits)
}

func (s *FrozenSuite) TestTimerStop() {
	var hits []int

	clock.AfterFunc(100, func() { hits = append(hits, 1) })
	t := clock.AfterFunc(100, func() { hits = append(hits, 2) })
	clock.AfterFunc(100, func() { hits = append(hits, 3) })
	clock.Advance(99)
	s.Require().Equal(0, len(hits))

	// When
	active1 := t.Stop()
	active2 := t.Stop()

	// Then
	s.Require().Equal(true, active1)
	s.Require().Equal(false, active2)
	clock.Advance(1)
	s.Require().Equal([]int{1, 3}, hits)
}

func (s *FrozenSuite) TestReset() {
	var hits []int

	t1 := clock.AfterFunc(100, func() { hits = append(hits, 1) })
	t2 := clock.AfterFunc(100, func() { hits = append(hits, 2) })
	clock.AfterFunc(100, func() { hits = append(hits, 3) })
	clock.Advance(99)
	s.Require().Equal(0, len(hits))

	// When
	active1 := t1.Reset(1) // Reset to the same time
	active2 := t2.Reset(7)

	// Then
	s.Require().Equal(true, active1)
	s.Require().Equal(true, active2)

	clock.Advance(1)
	s.Require().Equal([]int{3, 1}, hits)
	clock.Advance(5)
	s.Require().Equal([]int{3, 1}, hits)
	clock.Advance(1)
	s.Require().Equal([]int{3, 1, 2}, hits)
}

// Reset to the same time just puts the timer at the end of the trigger list
// for the date.
func (s *FrozenSuite) TestResetSame() {
	var hits []int

	t := clock.AfterFunc(100, func() { hits = append(hits, 1) })
	clock.AfterFunc(100, func() { hits = append(hits, 2) })
	clock.AfterFunc(100, func() { hits = append(hits, 3) })
	clock.AfterFunc(101, func() { hits = append(hits, 4) })
	clock.Advance(9)

	// When
	active := t.Reset(91)

	// Then
	s.Require().Equal(true, active)

	clock.Advance(90)
	s.Require().Equal(0, len(hits))
	clock.Advance(1)
	s.Require().Equal([]int{2, 3, 1}, hits)
}

func (s *FrozenSuite) TestTicker() {
	t := clock.NewTicker(100)

	clock.Advance(99)
	s.assertNotFired(t.C())
	clock.Advance(1)
	s.Require().Equal(<-t.C(), s.epoch.Add(100))
	clock.Advance(750)
	s.Require().Equal(<-t.C(), s.epoch.Add(200))
	clock.Advance(49)
	s.assertNotFired(t.C())
	clock.Advance(1)
	s.Require().Equal(<-t.C(), s.epoch.Add(900))

	t.Reset(200)
	clock.Advance(100)
	s.assertNotFired(t.C())
	clock.Advance(100)
	s.Require().Equal(<-t.C(), s.epoch.Add(1100))

	t.Stop()
	clock.Advance(300)
	s.assertNotFired(t.C())
}

func (s *FrozenSuite) TestTickerZero() {
	defer func() {
		_ = recover()
	}()

	clock.NewTicker(0)
	s.Fail("Should panic")
}

func (s *FrozenSuite) TestTick() {
	ch := clock.Tick(100)

	clock.Advance(99)
	s.assertNotFired(ch)
	clock.Advance(1)
	s.Require().Equal(<-ch, s.epoch.Add(100))
	clock.Advance(750)
	s.Require().Equal(<-ch, s.epoch.Add(200))
	clock.Advance(49)
	s.assertNotFired(ch)
	clock.Advance(1)
	s.Require().Equal(<-ch, s.epoch.Add(900))
}

func (s *FrozenSuite) TestTickZero() {
	ch := clock.Tick(0)
	s.Require().Nil(ch)
}

func (s *FrozenSuite) TestNewStoppedTimer() {
	t := clock.NewStoppedTimer()

	// When/Then
	select {
	case <-t.C():
		s.Fail("Timer should not have fired")
	default:
	}
	s.Require().Equal(false, t.Stop())
}

func (s *FrozenSuite) TestWait4Scheduled() {
	clock.After(100 * clock.Millisecond)
	clock.After(100 * clock.Millisecond)
	s.Require().Equal(false, clock.Wait4Scheduled(3, 0))

	startedCh := make(chan struct{})
	resultCh := make(chan bool)
	go func() {
		close(startedCh)
		resultCh <- clock.Wait4Scheduled(3, 5*clock.Second)
	}()
	// Allow some time for waiter to be set and start waiting for a signal.
	<-startedCh
	time.Sleep(50 * clock.Millisecond)

	// When
	clock.After(100 * clock.Millisecond)

	// Then
	s.Require().Equal(true, <-resultCh)
}

// If there is enough timers scheduled already, then a shortcut execution path
// is taken and Wait4Scheduled returns immediately.
func (s *FrozenSuite) TestWait4ScheduledImmediate() {
	clock.After(100 * clock.Millisecond)
	clock.After(100 * clock.Millisecond)
	// When/Then
	s.Require().Equal(true, clock.Wait4Scheduled(2, 0))
}

func (s *FrozenSuite) TestSince() {
	s.Require().Equal(clock.Duration(0), clock.Since(clock.Now()))
	s.Require().Equal(-clock.Millisecond, clock.Since(clock.Now().Add(clock.Millisecond)))
	s.Require().Equal(clock.Millisecond, clock.Since(clock.Now().Add(-clock.Millisecond)))
}

func (s *FrozenSuite) TestUntil() {
	s.Require().Equal(clock.Duration(0), clock.Until(clock.Now()))
	s.Require().Equal(clock.Millisecond, clock.Until(clock.Now().Add(clock.Millisecond)))
	s.Require().Equal(-clock.Millisecond, clock.Until(clock.Now().Add(-clock.Millisecond)))
}

func (s *FrozenSuite) assertHits(got <-chan int, want []int) {
	for i, w := range want {
		var g int
		select {
		case g = <-got:
		case <-time.After(100 * time.Millisecond):
			s.Failf("Missing hit", "want=%v", w)
			return
		}
		s.Require().Equal(w, g, "Hit #%d", i)
	}
	for {
		select {
		case g := <-got:
			s.Failf("Unexpected hit", "got=%v", g)
		default:
			return
		}
	}
}

func (s *FrozenSuite) assertNotFired(ch <-chan time.Time) {
	select {
	case <-ch:
		s.Fail("Premature fire")
	default:
	}
}
