package pool

import (
	"net"
	"sync"
	"time"
)

type (
	// workersPool with connections
	workersPool struct {
		workers          []*worker
		workersCount     int32
		workersBusyCount int32
		workersChn       chan Connection
		workersMux       *sync.RWMutex
	}

	Connection struct {
		Conn     net.Conn
		ConnType int
		Body     []byte
		BodyLen  int
	}
)

const (
	CONN_TCP = iota
	CONN_UNIX
	CONN_WS
	CONN_HTTP
)

var (
	defaultBufferSize      = 5120                      // 5KB
	defaultClientsDeadline = time.Duration(3000000000) // 3s
	emptyByte              = make([]byte, 0)
)

func checkConfig(config ManagerConfig) ManagerConfig {
	if config.BufferSize < 1 {
		config.BufferSize = defaultBufferSize
	}
	if config.ClientsDeadline < 1 {
		config.ClientsDeadline = defaultClientsDeadline
	}

	return config
}
