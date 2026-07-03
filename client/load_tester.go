package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"sync/atomic" // Added for local telemetry
	"time"
)

// Shared Constants
const (
	Blocksize  = 1024
	NoOfblocks = 10000
)

func main() {
	serverAddr := flag.String("addr", "127.0.0.1", "IP addr of remote vault")
	serverPort := flag.Int("port", 8080, "Port of remote vault")
	maxPeers   := flag.Int("peers", 10, "No of peers simulating GPU streams")
	opsPerPeer := flag.Int("ops", 100, "no. ops per peer")
	
	// NEW: Flag for targeted testing modes (distributed is the default)
	testMode   := flag.String("mode", "distributed", "Test strategy: 'distributed' or 'contended'")
	flag.Parse()

	target := fmt.Sprintf("%s:%v", *serverAddr, *serverPort)
	fmt.Printf("Starting network benchmark [%s mode] against %s (Active peers: %d)\n", *testMode, target, *maxPeers)

	var wg sync.WaitGroup
	var totalStores uint64
	var totalHits uint64
	var totalMisses uint64
	var failedConns uint64

	startTime := time.Now()

	for i := 0; i < *maxPeers; i++ {
		wg.Add(1)
		go func(peerID int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(peerID)))

			conn, err := net.Dial("tcp", target)
			if err != nil {
				atomic.AddUint64(&failedConns, 1)
				return
			}
			defer conn.Close()

			payload := make([]byte, Blocksize)
			rng.Read(payload)

			for op := 0; op < *opsPerPeer; op++ {
				// State 0=STORE, 1=FETCH, 2=SWAP. 
				// For clear metrics, we will alternate between STORE and FETCH/SWAP
				state := uint8(rng.Intn(3)) 
				
				var fkey, skey uint32
				if *testMode == "contended" {
					// High Contention Mode: Every peer hammers keys that hash to Bucket 0 exclusively
					fkey = uint32(rng.Intn(4)) * 256
					skey = uint32(rng.Intn(4)) * 256 + 256
				} else {
					// Distributed Mode (Default): Keys are scattered uniformly across all shards
					fkey = uint32(rng.Intn(NoOfblocks))
					skey = uint32(rng.Intn(NoOfblocks))
				}

				if fkey == skey {
					skey = (fkey + 1) % NoOfblocks
				}

				var opErr error
				switch state {
				case Storeflag:
					opErr = executestore(conn, skey, payload)
					if opErr == nil {
						atomic.AddUint64(&totalStores, 1)
					}
				case Fetchflag:
					_, opErr = executefetch(conn, fkey)
					if opErr == nil {
						atomic.AddUint64(&totalHits, 1)
					} else if opErr.Error() == "Error: Cache miss" || opErr.Error() == "error: block cache miss" {
						atomic.AddUint64(&totalMisses, 1)
						continue // It's okay to miss, we just track it
					}
				case Swapflag:
					// Treat swap hits as hits for simplicity in this reporter
					_, opErr = executeswap(conn, fkey, skey, payload)
					if opErr == nil {
						atomic.AddUint64(&totalHits, 1) 
					} else if opErr.Error() == "Error: Cache miss" || opErr.Error() == "error: block cache miss" {
						atomic.AddUint64(&totalMisses, 1)
						continue
					}
				}

				if opErr != nil {
					log.Printf("[Peer %d] Op %d Fatal Network Fault: %v", peerID, op, opErr)
					return // End this peer session on network fault
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)
	actualOps := atomic.LoadUint64(&totalStores) + atomic.LoadUint64(&totalHits) + atomic.LoadUint64(&totalMisses)

	fmt.Println("\n === FINAL CLIENT PERFORMANCE SUMMARY REPORT ===")
	fmt.Printf("  Execution Time      : %v\n", duration)
	fmt.Printf("  Connections Refused : %d\n", atomic.LoadUint64(&failedConns))
	fmt.Printf("  Total STORES Saved  : %d\n", atomic.LoadUint64(&totalStores))
	fmt.Printf("  Total FETCH/SWAP Hits : %d\n", atomic.LoadUint64(&totalHits))
	fmt.Printf("  Total FETCH/SWAP Misses : %d\n", atomic.LoadUint64(&totalMisses))
	fmt.Printf("  Total Operations    : %d\n", actualOps)
	if actualOps > 0 {
		fmt.Printf("  System Throughput   : %.2f ops/sec\n", float64(actualOps)/duration.Seconds())
	} else {
		fmt.Printf("  System Throughput   : 0.00 ops/sec (Server offline)\n")
	}
	fmt.Println("=================================================")
}