package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"errors"
	"log"
	"net"
	"os"         // Added for signal handling
	"os/signal" // Added for signal handling
	"sync/atomic" // Added for reading counters
	"syscall"   // Added for signal handling
)

func main() {
	bindaddr := flag.String("addr", "127.0.0.1", "IP addr to bind to server")
	bindport := flag.Int("port", 8080, "TCP port no. to listen to")
	flag.Parse()

	vault := NewVault()
	addr := fmt.Sprintf("%s:%d", *bindaddr, *bindport)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Fatal Error: failed to bind network socket to %s: %v", addr, err)
	}
	defer listener.Close()

	fmt.Printf("KV cache vault listening on %s ->\n", addr)

	// NEW: Intercept Ctrl+C to print telemetry report
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\n\n==========================================")
		fmt.Println("       FINAL SERVER REPORT   ")
		fmt.Println("==============================================")
		fmt.Printf("  Total STORE Operations : %d\n", atomic.LoadUint64(&vault.StoreCount))
		fmt.Printf("  Total FETCH Hits       : %d\n", atomic.LoadUint64(&vault.FetchHit))
		fmt.Printf("  Total FETCH Misses     : %d\n", atomic.LoadUint64(&vault.FetchMiss))
		fmt.Printf("  Total SWAP Hits        : %d\n", atomic.LoadUint64(&vault.SwapHit))
		fmt.Printf("  Total SWAP Misses      : %d\n", atomic.LoadUint64(&vault.SwapMiss))
		total := atomic.LoadUint64(&vault.StoreCount) + atomic.LoadUint64(&vault.FetchHit) + atomic.LoadUint64(&vault.FetchMiss) + atomic.LoadUint64(&vault.SwapHit) + atomic.LoadUint64(&vault.SwapMiss)
		fmt.Printf("  Aggregated Operations  : %d\n", total)
		os.Exit(0)
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Connection Error: Failed to receive connection: %v", err)
			continue
		}
		go handleclient(conn, vault)
	}
}

// Optimized handleclient (removed heavy connection logs per op, just track start/end)
func handleclient(conn net.Conn, vault *Vault) {
	defer conn.Close()
	// remoteAddr := conn.RemoteAddr().String()
	// log.Printf("[Server] New connection established: %s", remoteAddr) // Silenced for noise

	for {
		req, err := Parserequest(conn)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			break
		}
		switch req.Flag {
		case Storeflag:
			vault.Store(req.Skey, req.Value)
			if _, err := conn.Write([]byte{1}); err != nil {
				return
			}

		case Fetchflag:
			fvalue, err := vault.Fetch(req.Fkey)
			if err != nil {
				conn.Write([]byte{0}) // 0 = Cache Miss
				continue
			}
			// Send minimal header + payload
			conn.Write([]byte{1}) // 1 = Cache Hit
			binary.Write(conn, binary.BigEndian, uint32(len(fvalue)))
			if _, err := conn.Write(fvalue); err != nil {
				return
			}

		case Swapflag:
			swapvalue, err := vault.Swap(req.Fkey, req.Skey, req.Value)
			if err != nil {
				conn.Write([]byte{0})
				continue
			}
			conn.Write([]byte{1})
			binary.Write(conn, binary.BigEndian, uint32(len(swapvalue)))
			if _, err := conn.Write(swapvalue); err != nil {
				return
			}
		}
	}
}