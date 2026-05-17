package server

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
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
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service": "OCP DHT Bootstrap Node",
		"version": "1.0.0",
		"status":  "running",
	})
}

func (s *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
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
	json.NewEncoder(w).Encode(map[string]interface{}{
		"peers": s.node.GetPeers(),
		"count": s.node.PeerCount(),
	})
}

func (s *HTTPServer) handleRecords(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count": s.node.RecordCount(),
	})
}

// handleWebSocket performs a manual WebSocket upgrade using only
// the standard library then handles OCP DHT messages.
func (s *HTTPServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		http.Error(w, "expected websocket upgrade", http.StatusBadRequest)
		return
	}

	clientKey := r.Header.Get("Sec-Websocket-Key")
	if clientKey == "" {
		http.Error(w, "missing Sec-WebSocket-Key", http.StatusBadRequest)
		return
	}

	acceptKey := computeAcceptKey(clientKey)

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijack not supported", http.StatusInternalServerError)
		return
	}
	conn, bufrw, err := hj.Hijack()
	if err != nil {
		log.Printf("[ws] hijack error: %v", err)
		return
	}

	resp := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + acceptKey + "\r\n\r\n"
	if _, err := conn.Write([]byte(resp)); err != nil {
		conn.Close()
		return
	}

	peerAddr := r.RemoteAddr
	s.node.AddPeer(peerAddr)
	log.Printf("[ws] new peer connected: %s", peerAddr)
	defer func() {
		s.node.RemovePeer(peerAddr)
		conn.Close()
		log.Printf("[ws] peer disconnected: %s", peerAddr)
	}()

	go s.writePump(conn)
	s.readPump(conn, bufrw, peerAddr)
}

func (s *HTTPServer) readPump(conn net.Conn, r *bufio.ReadWriter, peer string) {
	for {
		conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		msg, err := readFrame(r.Reader)
		if err != nil {
			return
		}
		s.handleDHTMessage(conn, msg, peer)
	}
}

func (s *HTTPServer) writePump(conn net.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if err := writeFrame(conn, []byte(`{"type":"PING"}`)); err != nil {
			return
		}
	}
}

func (s *HTTPServer) handleDHTMessage(conn net.Conn, data []byte, peer string) {
	var msg map[string]interface{}
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}

	msgType, _ := msg["type"].(string)
	log.Printf("[ws] message from %s: type=%s", peer, msgType)

	switch msgType {
	case "PING":
		writeFrame(conn, []byte(`{"type":"PONG"}`))

	case "STORE":
		key, _ := msg["key"].(string)
		value, _ := msg["value"].(string)
		ttlFloat, _ := msg["ttl"].(float64)
		if key != "" && value != "" {
			ttl := time.Duration(ttlFloat) * time.Second
			if ttl <= 0 {
				ttl = 3600 * time.Second
			}
			s.node.Store().Store(key, []byte(value), ttl)
			log.Printf("[dht] stored key=%s ttl=%v", key, ttl)
			resp, _ := json.Marshal(map[string]string{
				"type": "STORED",
				"key":  key,
			})
			writeFrame(conn, resp)
		}

	case "FIND_VALUE":
		key, _ := msg["key"].(string)
		if key != "" {
			valBytes, found := s.node.Store().FindValue(key)
			if found {
				resp, _ := json.Marshal(map[string]string{
					"type":  "FOUND_VALUE",
					"key":   key,
					"value": string(valBytes),
				})
				writeFrame(conn, resp)
			} else {
				resp, _ := json.Marshal(map[string]interface{}{
					"type": "NOT_FOUND",
					"key":  key,
				})
				writeFrame(conn, resp)
			}
		}

	case "FIND_NODE":
		id, _ := msg["id"].(string)
		resp, _ := json.Marshal(map[string]interface{}{
			"type":       "FOUND_NODES",
			"node_id":    s.node.IDHex(),
			"peers":      s.node.GetPeers(),
			"queried_id": id,
		})
		writeFrame(conn, resp)
	}
}

// computeAcceptKey computes the Sec-WebSocket-Accept header value per RFC 6455.
func computeAcceptKey(clientKey string) string {
	const magic = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	h := sha1.New()
	h.Write([]byte(clientKey + magic))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// writeFrame writes an unmasked server WebSocket text frame.
func writeFrame(conn net.Conn, data []byte) error {
	n := len(data)
	var header []byte
	header = append(header, 0x81) // FIN + text opcode
	switch {
	case n <= 125:
		header = append(header, byte(n))
	case n <= 65535:
		header = append(header, 126, byte(n>>8), byte(n))
	default:
		header = append(header, 127, 0, 0, 0, 0,
			byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
	}
	_, err := conn.Write(append(header, data...))
	return err
}

// readFrame reads a masked client WebSocket frame.
func readFrame(r *bufio.Reader) ([]byte, error) {
	b0, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	opcode := b0 & 0x0F
	if opcode == 8 {
		return nil, fmt.Errorf("close frame")
	}

	b1, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	masked := b1&0x80 != 0
	payloadLen := int(b1 & 0x7F)

	if payloadLen == 126 {
		var ext [2]byte
		if _, err := r.Read(ext[:]); err != nil {
			return nil, err
		}
		payloadLen = int(ext[0])<<8 | int(ext[1])
	} else if payloadLen == 127 {
		var ext [8]byte
		if _, err := r.Read(ext[:]); err != nil {
			return nil, err
		}
		payloadLen = 0
		for i := 0; i < 8; i++ {
			payloadLen = payloadLen<<8 | int(ext[i])
		}
	}

	var maskKey [4]byte
	if masked {
		if _, err := r.Read(maskKey[:]); err != nil {
			return nil, err
		}
	}

	payload := make([]byte, payloadLen)
	for total := 0; total < payloadLen; {
		n, err := r.Read(payload[total:])
		total += n
		if err != nil {
			return nil, err
		}
	}

	if masked {
		for i := range payload {
			payload[i] ^= maskKey[i%4]
		}
	}

	// Respond to ping frames automatically.
	if opcode == 9 {
		return []byte(`{"type":"PONG"}`), nil
	}

	return payload, nil
}
