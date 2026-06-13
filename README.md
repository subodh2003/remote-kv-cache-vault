# Remote KV Cache Vault

A highly concurrent, sharded in-memory key-value cache vault featuring safe cross-bucket `SWAP` transactions, strict lock-ordering matrix configurations, and zero global lock contention. This engine is optimized for handling high-throughput mock processing streams natively across separate bare-metal hardware systems using Go's standard net package.

## Project Architecture

```text
remote-kv-cache-vault/
├── go.mod
├── server/
│   ├── main.go           # Server entry point and client handling loop
│   ├── engine.go         # Sharded memory vault with sync.RWMutex mapping
│   └── protocol.go       # Binary protocol frame deserialization
└── client/
    ├── load_tester.go    # Parallel multi-peer load driver
    └── encoding.go       # Protocol frame encoders
```
## How to run the server
1. Start the server first ->
```bash
$ go run server/*.go -addr 0.0.0.0 -port <default 8080>
```
3. Start the client ->
```bash
$ go run client/*.go -addr <SERVER_LAN_IP> -port <server port> -peers <default 50> -ops <default 100>
```

