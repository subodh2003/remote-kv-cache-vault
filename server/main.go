package main

import (
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
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

	fmt.Printf("KV cache vault listening on %s ->", addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Connection Error: Failed to receive connection: %v", err)
			continue
		}
		go handleclient(conn, vault)
	}
}

func handleclient(conn net.Conn, vault *Vault) {
	defer conn.Close()

	for {
		req, err := Parserequest(conn)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			log.Printf("Request Error: request identification failed: %v", err)
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
				conn.Write([]byte{0})
				return
			}
			responsebuf := make([]byte, 5)
			responsebuf[0] = 1
			binary.BigEndian.PutUint32(responsebuf[1:5], uint32(len(fvalue)))

			if _, err := conn.Write(responsebuf); err != nil {
				return
			}
			if _, err := conn.Write(fvalue); err != nil {
				return
			}

		case Swapflag:
			swapvalue, err := vault.Swap(req.Fkey, req.Skey, req.Value)
			if err != nil {
				conn.Write([]byte{0})
				return
			}
			responsebuf := make([]byte, 5)
			responsebuf[0] = 1
			binary.BigEndian.PutUint32(responsebuf[1:5], uint32(len(swapvalue)))

			if _, err := conn.Write(responsebuf); err != nil {
				return
			}
			if _, err := conn.Write(swapvalue); err != nil {
				return
			}
		}
	}
}
