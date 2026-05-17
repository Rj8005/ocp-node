# OCP DHT Node

A Kademlia-based DHT node for the Open Compute Protocol network, written in pure Go (standard library only).

## Architecture

```
ocp-node/
├── main.go           — entry point: node init, bootstrap, signal handling
├── dht/
│   ├── node.go       — NodeID, XOR distance, k-bucket routing table
│   ├── routing.go    — FindClosest, UpdateRoutingTable, bucket refresh
│   └── store.go      — key/value store with TTL expiry
└── server/
    └── websocket.go  — RFC 6455 WebSocket server, OCP message dispatcher
```

## Running

```bash
go run .
```

The node listens on **port 5000** (WebSocket at `/ws`, health check at `/health`).

## WebSocket Protocol

All messages are JSON. Send to `/ws`.

### FIND_NODE
```json
{"type":"FIND_NODE","id":"<hex-node-id-or-any-string>"}
```
Response:
```json
{"type":"RESPONSE","nodes":[{"id":"...","address":"..."}]}
```

### STORE
```json
{"type":"STORE","key":"mykey","value":"myvalue","ttl":3600}
```
`ttl` is in seconds (default 86400). Response:
```json
{"type":"RESPONSE"}
```

### FIND_VALUE
```json
{"type":"FIND_VALUE","key":"mykey"}
```
Response (found):
```json
{"type":"RESPONSE","key":"mykey","value":"myvalue"}
```
Response (not found — returns closest nodes):
```json
{"type":"RESPONSE","nodes":[...]}
```

## Bootstrap Nodes

Edit the `bootstrapNodes` slice in [main.go](main.go) before deployment:

```go
var bootstrapNodes = []string{
    "peer1.example.com:5000",
    "peer2.example.com:5000",
}
```

## Configuration

| Constant | Default | Description |
|---|---|---|
| `K` | 20 | k-bucket size |
| `Alpha` | 3 | parallelism factor |
| `BucketLen` | 160 | number of buckets (bits in ID) |
| `IDLen` | 20 | node ID length in bytes (SHA-1) |
