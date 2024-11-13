package wait

import "sync"

// FanOut spawns a new go-routine each time `Run()` is called until `size` is reached,
// subsequent calls to `Run()` will block until previously `Run()` routines have completed.
// This allows the user to control how many routines will run simultaneously. `Wait()` then
// collects any errors from the routines once they have all completed.
type FanOut struct {
	errChan chan error
	size    chan bool
	errs    MultiError
	wg      sync.WaitGroup
}

func NewFanOut(size int) *FanOut {
	// They probably want no concurrency
	if size == 0 {
		size = 1
	}

	pool := FanOut{
		errChan: make(chan error, size),
		size:    make(chan bool, size),
	}
	pool.start()
	return &pool
}

func (p *FanOut) start() {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		for err := range p.errChan {
			p.errs = append(p.errs, err)
		}
	}()
}

// Run a new routine with an optional data value
func (p *FanOut) Run(cb func() error) {
	if p.errChan == nil {
		panic("must call wait.NewFanOut() first")
	}

	p.size <- true
	go func() {
		err := cb()
		if err != nil {
			p.errChan <- err
		}
		<-p.size
	}()
}

// Wait for all the routines to complete and return any errors
func (p *FanOut) Wait() error {
	// Wait for all the routines to complete
	for i := 0; i < cap(p.size); i++ {
		p.size <- true
	}
	// Close the err channel
	if p.errChan != nil {
		close(p.errChan)
	}

	// Wait until the error collector routine is complete
	p.wg.Wait()

	defer func() {
		p.errs = nil
	}()

	// If there are no errors
	if len(p.errs) == 0 {
		return nil
	}
	return p.errs
}
