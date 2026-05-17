package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Rj8005/ocp-node/dht"
)

type HTTPServer struct {
	node      *dht.Node
	startTime time.Time
	port      int
}

func NewHTTPServer(node *dht.Node, port int) *HTTPServer {
	return &HTTPServer{
		node:      node,
		startTime: time.Now(),
		port:      port,
	}
}

func (s *HTTPServer) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/peers", s.handlePeers)
	mux.HandleFunc("/records", s.handleRecords)
	mux.HandleFunc("/ws", s.handleWebSocket)
	mux.HandleFunc("/", s.handleRoot)

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("[http] listening on %s", addr)
	return http.ListenAndServe(addr, mux)
}

func (s *HTTPServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service": "OCP DHT Bootstrap Node",
		"version": "1.0.0",
		"status":  "running",
		"endpoints": map[string]string{
			"health":    "/health",
			"peers":     "/peers",
			"records":   "/records",
			"websocket": "/ws",
		},
	})
}

func (s *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"node_id": s.node.IDHex(),
		"address": s.node.Address(),
		"uptime":  int(time.Since(s.startTime).Seconds()),
		"peers":   s.node.PeerCount(),
		"records": s.node.RecordCount(),
	})
}

func (s *HTTPServer) handlePeers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"peers": s.node.GetPeers(),
		"count": s.node.PeerCount(),
	})
}

func (s *HTTPServer) handleRecords(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count": s.node.RecordCount(),
	})
}

func (s *HTTPServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// WebSocket upgrade handled by ws server
	// This is just a placeholder to avoid 404
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "use wss:// for WebSocket connections",
	})
}
