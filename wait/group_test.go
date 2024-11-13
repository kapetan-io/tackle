package wait_test

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kapetan-io/tackle/wait"
)

func TestRun(t *testing.T) {
	var wg wait.Group

	errs := []error{
		errors.New("error 1"),
		errors.New("error 2"),
	}

	for _, err := range errs {
		wg.Run(func() error {
			time.Sleep(time.Nanosecond * 50)
			return err
		})
	}

	require.Error(t, wg.Wait())
	assert.Equal(t, 2, len(errs))
	assert.Contains(t, errs, errs[0])
	assert.Contains(t, errs, errs[1])
}

func TestGo(t *testing.T) {
	var wg wait.Group
	result := make(chan struct{})

	wg.Go(func() {
		time.Sleep(time.Nanosecond * 500)
		result <- struct{}{}
	})

	wg.Go(func() {
		time.Sleep(time.Nanosecond * 50)
		result <- struct{}{}
	})

	for i := 0; i < 2; {
		select {
		case <-result:
			i++
		case <-time.After(time.Second):
			t.Fatalf("waited to long for Go() to run")
		}
	}
	assert.NoError(t, wg.Wait())
}

func TestLoop(t *testing.T) {
	pipe := make(chan int32)
	var wg wait.Group
	var count int32

	wg.Loop(func() bool {
		inc, ok := <-pipe
		if !ok {
			return false
		}
		atomic.AddInt32(&count, inc)
		return true
	})

	// Feed the loop some numbers and close the pipe
	pipe <- 1
	pipe <- 5
	pipe <- 10
	close(pipe)

	// Wait for the routine to end
	// no error collection when using Loop()
	assert.NoError(t, wg.Wait())
	assert.Equal(t, int32(16), count)
}

func TestUntil(t *testing.T) {
	pipe := make(chan int32)
	var wg wait.Group
	var count int32

	wg.Until(func(done chan struct{}) bool {
		select {
		case inc := <-pipe:
			atomic.AddInt32(&count, inc)
		case <-done:
			return false
		}
		return true
	})

	wg.Until(func(done chan struct{}) bool {
		select {
		case inc := <-pipe:
			atomic.AddInt32(&count, inc)
		case <-done:
			return false
		}
		return true
	})

	// Feed the loop some numbers
	pipe <- 1
	pipe <- 5
	pipe <- 10

	// Shutdown the Until loop
	wg.Stop()
	assert.Equal(t, int32(16), count)
}
