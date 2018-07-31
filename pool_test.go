package pool_test

import (
	"fmt"
	"log"
	"net"
	"os"
	"testing"

	pool "github.com/maclisowski/pool-manager"
)

var (
	conn              net.Conn
	poolManager       pool.Manager
	poolManagerConfig = pool.ManagerConfig{
		MinWorkersCount: 0,
	}
	poolCallback = func(conn net.Conn, body []byte, bodyLen int) {
		conn.Write([]byte(`OK`))
	}
)

// go test -run='^$' -bench=. -benchmem
func init() {
	poolManager = pool.CreateManager(poolManagerConfig, poolCallback)
	poolManager.Start()

	go simpleTCPServer()
}

func TestCreateManager(t *testing.T) {
	_ = pool.CreateManager(poolManagerConfig, poolCallback)
}

func TestStartPoolManager(t *testing.T) {
	manager := pool.CreateManager(poolManagerConfig, poolCallback)
	manager.Start()
}

func TestPut(t *testing.T) {
	dial, dialErr := net.Dial("tcp", "127.0.0.1:6263")
	if dialErr != nil {
		t.Error(dialErr)
	}

	_, err := dial.Write([]byte(`HI!`))
	if err != nil {
		t.Error(err)
	}
}

func Benchmark_PUT(b *testing.B) {
	b.SetBytes(1)
	b.ReportAllocs()

	dial, dialErr := net.Dial("tcp", "127.0.0.1:6263")
	if dialErr != nil {
		b.Error(dialErr)
	}

	for n := 0; n < b.N; n++ {
		_, err := dial.Write([]byte(`HI!`))
		if err != nil {
			b.Error(err)
		}
	}
}

func Benchmark_Put_P2(b *testing.B) {
	b.SetBytes(1)
	b.ReportAllocs()
	b.SetParallelism(2)

	b.RunParallel(func(pb *testing.PB) {
		dial, dialErr := net.Dial("tcp", "127.0.0.1:6263")
		if dialErr != nil {
			b.Error(dialErr)
		}

		for pb.Next() {
			_, err := dial.Write([]byte(`HI!`))
			if err != nil {
				b.Error(err)
			}
		}

	})
}

func Benchmark_Put_P4(b *testing.B) {
	b.SetBytes(1)
	b.ReportAllocs()
	b.SetParallelism(4)

	b.RunParallel(func(pb *testing.PB) {
		dial, dialErr := net.Dial("tcp", "127.0.0.1:6263")
		if dialErr != nil {
			b.Error(dialErr)
		}

		for pb.Next() {
			_, err := dial.Write([]byte(`HI!`))
			if err != nil {
				b.Error(err)
			}
		}

	})
}

func Benchmark_Put_P10(b *testing.B) {
	b.SetBytes(1)
	b.ReportAllocs()
	b.SetParallelism(10)

	b.RunParallel(func(pb *testing.PB) {
		dial, dialErr := net.Dial("tcp", "127.0.0.1:6263")
		if dialErr != nil {
			b.Error(dialErr)
		}

		for pb.Next() {
			_, err := dial.Write([]byte(`HI!`))
			if err != nil {
				b.Error(err)
			}
		}

	})
}

func Benchmark_Put_P20(b *testing.B) {
	b.SetBytes(1)
	b.ReportAllocs()
	b.SetParallelism(20)

	b.RunParallel(func(pb *testing.PB) {
		dial, dialErr := net.Dial("tcp", "127.0.0.1:6263")
		if dialErr != nil {
			b.Error(dialErr)
		}

		for pb.Next() {
			_, err := dial.Write([]byte(`HI!`))
			if err != nil {
				b.Error(err)
			}
		}

	})
}

func simpleTCPServer() {
	listener, err := net.Listen("tcp", "127.0.0.1:6263")
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}

		errPut := poolManager.Put(conn)
		if errPut != nil {
			conn.Write([]byte(`ERROR_SERVER_BUSY`))
			conn.Close()
			continue
		}
	}
}
