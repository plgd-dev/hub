package main

import (
	"flag"
	"net"
	"sync"

	"github.com/plgd-dev/hub/v2/pkg/log"
)

func main() {
	numConn := flag.Int("numConn", 1, "number of connections")
	addr := flag.String("addr", "127.0.0.1:7000", "address to connect to")
	flag.Parse()
	var wg sync.WaitGroup
	wg.Add(*numConn)
	for i := 0; i < *numConn; i++ {
		conn, err := net.Dial("tcp", *addr)
		if err != nil {
			log.Fatalf("cannot dial: %v", err)
		}
		go func(conn net.Conn) {
			defer wg.Done()
			defer conn.Close()
			buf := make([]byte, 1024)
			for {
				n, err := conn.Read(buf)
				if err != nil {
					log.Errorf("cannot read: %v", err)
					return
				}
				log.Debugf("read: %v", string(buf[:n]))
			}
		}(conn)
	}
	wg.Wait()
}
