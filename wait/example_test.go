package wait_test

import (
	"fmt"
	"github.com/kapetan-io/tackle/wait"
	"testing"
	"time"
)

func TestExampleGo(t *testing.T) {
	items := []int{0, 1, 3, 4, 5}

	var wg wait.Group
	for _, item := range items {
		wg.Go(func() {
			// Do something with 'item'
			fmt.Printf("Item: %d\n", item)
			time.Sleep(time.Nanosecond * 50)
		})
	}

	_ = wg.Wait()
}

func TestExampleUntil(t *testing.T) {
	var wg wait.Group

	wg.Until(func(done chan struct{}) bool {
		select {
		case <-time.Tick(time.Second):
			// Do some periodic thing
		case <-done:
			return false
		}
		return true
	})

	// Close the done channel and wait for the routine to exit
	wg.Stop()
}

func TestExampleFanOut(t *testing.T) {
	fanOut := wait.NewFanOut(10)
	items := []int{0, 1, 3, 4, 5}

	for _, item := range items {
		fanOut.Run(func() error {
			fmt.Printf("Item: %d\n", item)
			time.Sleep(time.Nanosecond * 50)
			//return db.ExecuteQuery("insert into tbl (id, field) values (?, ?)",
			// item.Id, item.Field)
			return nil
		})
	}

	// Collect errors if any
	err := fanOut.Wait()
	if err != nil {
		panic(err)
	}
}
