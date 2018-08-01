package pool

import (
	"bufio"
	"errors"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type (
	// Manager responsible for managing pool of connections
	Manager struct {
		config      ManagerConfig
		callback    func(conn net.Conn, requestBody []byte, requestBodyLen int)
		pool        workersPool
		killCleaner chan bool
		isStop      bool
	}

	// ManagerConfig represents struct for configuration parameters
	ManagerConfig struct {
		Debug           bool          // if you want to log debug messages
		BufferSize      int           // buffer size when reading clients connections (in bytes), by default it is 5120
		MinWorkersCount int32         // min number of pool workers - can be 0
		MaxWorkersCount int32         // max number of pool workers - can be 0, it means then that there is no max number of workers
		ClientsDeadline time.Duration // interval from Now() in nanoseconds when connection will timeout if there is no activity, by default it is 3s
	}
)

// CreateManager creates new pool manager
func CreateManager(config ManagerConfig, callback func(conn net.Conn, requestBody []byte, requestBodyLen int)) Manager {

	return Manager{
		config:      checkConfig(config),
		callback:    callback,
		killCleaner: make(chan bool),
		isStop:      true,
		pool: workersPool{
			workersMux: new(sync.RWMutex),
			workersChn: make(chan net.Conn),
		},
	}
}

// Start fiers up workers and initialize pool
func (m *Manager) Start() error {
	if m.isStop == false {
		return errors.New("pool manager is already running")
	}
	m.isStop = false

	// start pool workers
	n := int32(0)
	for ; n < m.config.MinWorkersCount; n++ {
		m.spawnWorker()
	}
	m.pool.workersCount = m.config.MinWorkersCount

	go m.cleaner()

	return nil
}

// Put used to add new connection to the pool
func (m *Manager) Put(conn net.Conn) error {
	if m.config.Debug == true {
		log.Println("[pool]: Put new connection")
	}

	if m.isStop == true {
		return errors.New("pool manager is not running, please use Start() method to run")
	} else if m.config.MaxWorkersCount > 0 && m.pool.workersBusyCount >= m.config.MaxWorkersCount {
		return errors.New("can't put, pool is busy")
	} else if (m.config.MaxWorkersCount > 0 && m.pool.workersBusyCount >= m.pool.workersCount) || (m.config.MaxWorkersCount < 1 && m.pool.workersBusyCount >= m.config.MinWorkersCount) {
		m.spawnWorker()
	}
	atomic.AddInt32(&m.pool.workersBusyCount, 1)

	m.pool.workersChn <- m.setDeadline(conn, m.config.ClientsDeadline)
	return nil
}

// Config  return current pool manager config values. It will return ManagerConfig struct.
func (m *Manager) Config() ManagerConfig {
	return m.config
}

// SetConfig will set new configuration for pool manager
func (m *Manager) SetConfig(config ManagerConfig) {
	m.config = checkConfig(config)
}

// Stop will gracefully stop pool manager - will wait for all busy workers to finish their job and then will kill all of them
func (m *Manager) Stop() {
	m.isStop = true

	allKilled := make(chan bool)
	// kill all workers
	go func(allKiled chan bool, workers []*worker) {
		killed := 0
		toKill := len(workers)

		for {
			for _, worker := range workers {
				if worker.getState() != WorkerStateBusy {
					worker.shutdown()
					killed++
				}
			}

			if killed < toKill {
				continue
			}
			break
		}
		allKiled <- true
	}(allKilled, m.pool.workers)

	<-allKilled

	m.killCleaner <- true
}

func (m *Manager) spawnWorker() {
	worker := &worker{
		handler: m.worker,
		state:   WorkerStateInit,
		mux:     new(sync.Mutex),
		kill:    make(chan bool),
	}

	m.pool.workersMux.RLock()
	m.pool.workers = append(m.pool.workers, worker)
	m.pool.workersMux.RUnlock()

	atomic.AddInt32(&m.pool.workersCount, 1)

	go worker.handler(worker, m.pool.workersChn)
}

func (m *Manager) worker(w *worker, conns chan net.Conn) {

	workToDo := func(w *worker, conns chan net.Conn) bool {
		for {
			select {
			case <-w.kill:
				w.setState(WorkerStateKilled)
				return true
			case conn := <-conns:
				w.setState(WorkerStateBusy)

				errConn := m.checkConnection(conn)

				if errConn != nil {
					conn.Close()
					w.setState(WorkerStateIdle)
					continue
				}

				for {
					// pseudo rate limiter - this could be extended to allow to limit rate of reuqests
					time.Sleep(100 * time.Nanosecond)

					connBody, err := m.readConn(conn, m.config.ClientsDeadline)
					if err != nil {
						conn.Close()
						break
					}

					// send body request to callback
					m.callback(conn, connBody, len(connBody))
				}
				w.setState(WorkerStateIdle)
				atomic.AddInt32(&m.pool.workersBusyCount, -1)
			}
		}
	}

	workToDo(w, conns)
}

// cleaner runs every 2s
func (m *Manager) cleaner() {
	for {
		time.Sleep(2 * time.Second)

		if m.pool.workersCount > m.config.MinWorkersCount && m.pool.workersBusyCount < m.config.MinWorkersCount {
			toDelete := m.pool.workersCount - m.config.MinWorkersCount
			deleted := int32(0)

			i := len(m.pool.workers) - 1

			for ; i >= 0; i-- {
				if deleted >= toDelete {
					break
				}

				worker := m.pool.workers[i]

				if worker.state == WorkerStateIdle {
					worker.shutdown()

					m.pool.workers = append(m.pool.workers[:i], m.pool.workers[i+1:]...)

					atomic.AddInt32(&m.pool.workersCount, -1)
					deleted++
				}
			}
			continue
		}

		select {
		case <-m.killCleaner:
			return
		default:
			continue
		}
	}
}

func (m *Manager) readConn(client net.Conn, deadline time.Duration) ([]byte, error) {
	var (
		tmpBuffer  = make([]byte, m.config.BufferSize)
		clientData = make([]byte, 0)
		err        error
		n          int
	)

	// using bufo reader to limit syscalls
	buf := bufio.NewReader(client)
	for {
		n, err = buf.Read(tmpBuffer)
		if err != nil {
			return clientData, err
		}

		clientData = append(clientData, tmpBuffer[:n]...)
		if n < m.config.BufferSize {
			return clientData, nil
		}
	}
}

func (m *Manager) setDeadline(conn net.Conn, deadline time.Duration) net.Conn {
	conn.SetDeadline(time.Now().Add(deadline * time.Nanosecond))
	return conn
}

func (m *Manager) checkConnection(conn net.Conn) error {
	_, err := conn.Read(emptyByte)
	if err != nil {
		return err
	}

	return nil
}
