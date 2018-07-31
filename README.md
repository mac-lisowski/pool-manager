# pool-manager (v1.0.0) [![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](https://godoc.org/github.com/maclisowski/pool-manager)


<p align="center">
    <img src="https://user-images.githubusercontent.com/3139143/30776268-17b89fb6-a069-11e7-829a-0e9caf3b69bf.png" alt="drawing" width="200px" align="middle"/>
</p>
<br>
-- 

### Table of contents

* Methods
 * [CreateManager(config ManagerConfig, callback func(conn net.Conn, requestBody []byte, requestBodyLen int)) Manager](#pm-m-createmanager)
 * [func (m *Manager) Start() error](#pm-m-start)
 * [func (m *Manager) Put(conn net.Conn) error](#pm-m-put)
 * [func (m *Manager) Config() ManagerConfig](#pm-m-config)
 * [func (m *Manager) SetConfig(config ManagerConfig) ManagerConfig](#pm-m-setconfig)
 * [func (m *Manager) Stop()](#pm-m-stop)
* [Callback handler](#pm-callbackhandler)
* [ManagerConfig](#pm-managerconfig)
* [Example TCP Server implementation](#pm-example-tcp)

***
    
### Pool Manager
Pool Manager is responsible for managing pool of workers which are handling requests from `net.Conn` connections - accepting/rejecting connections, reading data sent by clients and then handling those data over to [callback](#pm-callbackhandler) handler as type `[]byte` where they can be processed more further depends on use case.

Pool manager can have defined min and max number of workers - where both are optional. In case where max is not defined - then pool will spawn as many workers as needed to handle all incoming connections.

-- 

#### <a name="pm-callbackhandler">Callback handler</a>

This handler method is custom method which implementation can depend on the use case. Method is invoked each time pool worker is done with reading connection data. 

Callback method implements this interface:<br>
`func(conn net.Conn, requestBody []byte, requestBodyLen int)`

Example:

```go
callback := func(conn net.Conn, requestBody []byte, requestBodyLen int) {
	if requestBodyLen > 0 {
		conn.Write([]byte(`OK`))
	} else {
		conn.Write([]byte(`ERROR`))
	}
}
```

--

#### <a name="pm-managerconfig">ManagerConfig</a>
Config is passed over during creation of new pool manager.

```go
// ManagerConfig represents struct for configuration parameters
ManagerConfig struct {
	Debug           bool          // if you want to log debug messages
	BufferSize      int           // buffer size when reading clients connections (in bytes), by default it is 5120
	MinWorkersCount int32         // min number of pool workers - can be 0
	MaxWorkersCount int32         // max number of pool workers - can be 0, it means then that there is no max number of workers
	ClientsDeadline time.Duration // interval from Now() in nanoseconds when connection will timeout if there is no activity, by default it is 3s
}
```

--

#### Methods


##### <a name="pm-m-createmanager">CreateManager(config ManagerConfig, callback func(conn net.Conn, requestBody []byte, requestBodyLen int)) Manager</a>

Use this method to create new connections pool manager. In order to create new manager, you have to pass config (`pool.ManagerConfig`) and [callback](#pm-callbackhandler) function which will be called each time pool worker has finished reading connection.

To start pool manager use method func(m *Manager) Start()

Pleae see example on how to create new manager object:

```go
managerConfig := pool.ManagerConfig{
	Debug:               true,
	BufferSize:          5120,
	MinWorkersCount:     5,
	MaxWorkersCount:     10,
} 

poolManager := pool.CreateManager(managerConfig)
```
<small>Ref: [ManagerConfig](#pm-managerconfig)</small>

-- 

##### <a name="pm-m-start">func (m *Manager) Start() error</a>
Use this method to run pool manager based on configuration passed over during creation of new manager. 
This method will create as many workers as specified in `ManagerConfig.MinWorkersCount` and will start helper worker which is responsible for killing unused workers if needed. 

Returns error if manager is already running.

--

##### <a name="pm-m-put">func (m *Manager) Put(conn net.Conn) error</a>
Use this method to put new client connection to the pool to be processed by one of workers. If you have defined min number of workers and they are not busy - connection will be passed over right away to a free worker, thanks for that connection can be processed faster beacuse we are not wasting time to spawn a new worker. 

In case where there is no min workers or min number of workers is busy (and max number of workers is higher than min) - manager will spawn new worker and will pass over connection to the worker. 

In case where max number of workers is defined and all workers are busy it will return an error `can't put, pool is busy`. On success error is equal `nil`.

Example:

```go
var conn net.Conn

errPut := poolManager.Put(conn)
if errPut != nil {
	conn.Write([]byte(`ERROR_SERVER_BUSY`))
	conn.Close()
}
```

--

##### <a name="pm-m-config">func (m *Manager) Config() ManagerConfig</a>
Use this method to return current pool manager config values. It will return [ManagerConfig](#pm-managerconfig) struct.

--

##### <a name="pm-m-setconfig">func (m *Manager) SetConfig(config ManagerConfig) ManagerConfig</a>
Use this method to set new config for pool manager. You can update configuration settings while pool manager is working - if for example you would like to adjust number of min workers.  It will return [ManagerConfig](#pm-managerconfig) struct.

--

##### <a name="pm-m-stop">func (m *Manager) Stop()</a>
Stop will gracefully shutdown pool manager - will wait for all busy workers to finish their job and then will kill all of them. It is safe to usu `Start()` again later in code in order to run pool manager again.

***

##### <a name="pm-example-tcp">Example TCP Server implementation</a>
Please check this example to see how you can tackle this pool manager to handle TCP connections:

```go
package main

import (
	"fmt"
	"log"
	"net"
	"os"
	
	pool "github.com/maclisowski/pool-manager"
)

func main() {
	poolManager := pool.CreateManager(pool.ManagerConfig{
		Debug:           true,
		BufferSize:      5120,
		MinWorkersCount: 5,
		MaxWorkersCount: 10,
	}, poolCallback)
	
	poolManager.Start()
	
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

func poolCallback(conn net.Conn, requestBody []byte, requestBodyLen int) {
	if requestBodyLen > 0 {
		conn.Write([]byte(`OK`))
	} else {
		conn.Write([]byte(`ERROR`))
	}
}
```

--

This package is under MIT License (MIT). Please see LICENSE file for more details.

Copyright (c) 2018 Maciej Lisowski

--
