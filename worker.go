package pool

import (
	"net"
	"sync"
)

type (
	// worker represents struct for worker used by pool manager
	worker struct {
		handler workerHandler
		mux     *sync.Mutex
		state   int
		kill    chan bool
	}

	// workerHandler represents interface for pool worker method
	workerHandler func(worker *worker, conns chan net.Conn)
)

const (
	// WorkerStateInit is set during initialization
	WorkerStateInit = iota
	// WorkerStateBusy is set when worker is currenlty processing data
	WorkerStateBusy
	// WorkerStateIdle is set when worker finished processing data and is waiting
	WorkerStateIdle
	// WorkerStateKilled is set when worker got signal to kill
	WorkerStateKilled
)

// kill sends signal for worker to kill
func (w *worker) shutdown() {
	w.kill <- true
}

// setState handles state transformations for pool worker
func (w *worker) setState(state int) {
	w.mux.Lock()
	defer w.mux.Unlock()

	w.state = state
}

// state returns current worker state
func (w *worker) getState() int {
	return w.state
}
