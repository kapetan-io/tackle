package wait_test

import (
	"github.com/kapetan-io/tackle/wait"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"slices"
	"testing"
)

func TestFanOut(t *testing.T) {
	f := wait.NewFanOut(5)
	ch := make(chan int, 20)

	for i := 0; i < 10; i++ {
		f.Run(func() error {
			//t.Logf("Concurrent: %d\n", i)
			ch <- i
			return nil
		})
	}
	require.NoError(t, f.Wait())
	close(ch)

	var results []int
	for v := range ch {
		results = append(results, v)
	}
	assert.Equal(t, 10, len(results))

	// Ensure no duplicate integers
	slices.Sort(results)
	assert.Equal(t, results, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})
}
