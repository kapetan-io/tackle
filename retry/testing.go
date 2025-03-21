package retry

import (
	"fmt"
	"net"
	"time"
)

type TestingT interface {
	Errorf(format string, args ...interface{})
	FailNow()
}

type TestResults struct {
	T        TestingT
	Failures []string
}

func (s *TestResults) Errorf(format string, args ...interface{}) {
	s.Failures = append(s.Failures, fmt.Sprintf(format, args...))
}

func (s *TestResults) FailNow() {
	s.Report(s.T)
	s.T.FailNow()
}

func (s *TestResults) Report(t TestingT) {
	for _, failure := range s.Failures {
		t.Errorf(failure)
	}
}

// UntilPass returns true if all the assertions in the provided callback eventually pass. Returns false if the
// there were some assertions that failed after the number of attempts is exceeded. This can be used to determine
// if the testing should proceed if the assertions in UntilPass() failed to eventually pass.
//
// UntilPass is compatible with both `assert` and `require` packages from `testify`. Use of `require` assertions
// will cause the callback to immediately fail, useful for avoiding panics from nil assertions. You can also
// call t.FailNow() to get the same behavior.
//
// UntilPass collects the results of each callback attempt using retry.TestResults. It only reports the assertion
// failures reported to retry.TestResults if the number of attempts is exceeded.
func UntilPass(t TestingT, attempts int, sleep time.Duration, callback func(t TestingT)) bool {
	results := TestResults{T: t}

	for i := 0; i < attempts; i++ {
		// Clear the failures before each attempt
		results.Failures = nil

		// Run the tests in the callback
		callback(&results)

		// If the test had no failures
		if len(results.Failures) == 0 {
			return true
		}
		time.Sleep(sleep)
	}
	// We have exhausted our attempts and should report the failures and exit
	results.Report(t)
	return false
}

// UntilConnect attempts to connect to the specified tcp address until either a successful connect or attempts
// are exhausted. It is useful to know the server spawned is up and listening for requests prior to a test running.
func UntilConnect(t TestingT, a int, d time.Duration, addr string) {
	for i := 0; i < a; i++ {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			continue
		}
		conn.Close()
		time.Sleep(d)
		return
	}
	t.Errorf("never connected to TCP server at '%s' after %d attempts", addr, a)
	t.FailNow()
}
