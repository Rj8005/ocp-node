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

## Running locally

```bash
go run .
```

The node listens on **port 5000** (WebSocket at `/ws`, health check at `/health`).

## Deployment — Render

Bootstrap nodes are hosted on [Render](https://render.com) as a Docker-based web service.

### Steps

1. Push this repo to GitHub.
2. In Render: **New → Web Service → Connect your repo**.
3. Set the following:
   - **Environment:** Docker
   - **Port:** 5000
   - **Instance type:** Free (or Starter for always-on)
4. Deploy. Render builds via the [Dockerfile](Dockerfile) and exposes the service over HTTPS.

### Always-on bootstrap node

Free-tier Render instances spin down after inactivity. For a reliable bootstrap node use a **Starter** (paid) instance, or set up a cron job / UptimeRobot to ping `/health` every 5 minutes.

### Updating bootstrap addresses

After deploying, copy the Render service URL (e.g. `ocp-bootstrap-1.onrender.com`) and update `main.go`:

```go
var bootstrapNodes = []string{
    "ocp-bootstrap-1.onrender.com:443",
    "ocp-bootstrap-2.onrender.com:443",
}
```

## WebSocket Protocol

All messages are JSON. Connect to `wss://<host>/ws`.

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

## Configuration

| Constant | Default | Description |
|---|---|---|
| `K` | 20 | k-bucket size |
| `Alpha` | 3 | parallelism factor |
| `BucketLen` | 160 | number of buckets (bits in ID) |
| `IDLen` | 20 | node ID length in bytes (SHA-1) |
