package main

import (
	// "errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"
)

func main() {
	serverAddr := flag.String("addr", "127.0.0.1", "IP addr of remote vault")
	serverPort := flag.Int("port", 8080, "Port of remote vault")
	maxPeers := flag.Int("peers", 50, "No of peers simulating gpu streams")
	opsPerPeer := flag.Int("ops", 100, "no. ops per peer")
	flag.Parse()

	target := fmt.Sprintf("%s:%v", *serverAddr, *serverPort)
	fmt.Printf("Starting network benchmark against %s (Active peers: %d)\n", target, *maxPeers)

	var wg sync.WaitGroup
	stime := time.Now()

	for i := 0; i < *maxPeers; i++ {
		wg.Add(1)
		go func(peerid int) {
			defer wg.Done()
			// thread safety for global rand function
			rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(peerid)))

			conn, err := net.Dial("tcp", target)
			if err != nil {
				log.Printf("[Peer %d] Network link initiation failed: %v", peerid, err)
				return
			}
			defer conn.Close()

			svalue := make([]byte, 1024*1024)
			rng.Read(svalue)
			for op := 0; op < *opsPerPeer; op++ {
				state := uint8(rng.Intn(3))
				fkey := uint32(rng.Intn(10000))
				skey := uint32(rng.Intn(10000))

				if fkey == skey {
					skey = (fkey + 1) % 10000
				}

				var err error
				switch state {
				case Storeflag:
					err = executestore(conn, skey, svalue)
				case Fetchflag:
					_, err = executefetch(conn, fkey)
				case Swapflag:
					_, err = executeswap(conn, fkey, skey, svalue)
				}

				if err != nil {
					// log.Printf("[Peer %d] Op %d failed: %v", peerid, op, err)
					//  // Abort this session on network fault
					continue
				}
			}

		}(i)
	}
	wg.Wait()
	delta := time.Since(stime)
	totalOps := (*maxPeers) * (*opsPerPeer)
	fmt.Printf("Benchmark finished. Processed %d operations across %d peers in %v (%.2f ops/sec)\n",
		totalOps, *maxPeers, delta, float64(totalOps)/delta.Seconds())

}
