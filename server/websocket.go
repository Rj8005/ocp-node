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
	"sync"
	"time"

	"github.com/Rj8005/ocp-node/dht"
)

// Message types for the OCP DHT protocol.
const (
	MsgFindNode  = "FIND_NODE"
	MsgStore     = "STORE"
	MsgFindValue = "FIND_VALUE"
	MsgResponse  = "RESPONSE"
	MsgError     = "ERROR"
)

type Message struct {
	Type    string          `json:"type"`
	ID      string          `json:"id,omitempty"`
	Key     string          `json:"key,omitempty"`
	Value   string          `json:"value,omitempty"`
	TTL     int             `json:"ttl,omitempty"`
	Nodes   []ContactJSON   `json:"nodes,omitempty"`
	Error   string          `json:"error,omitempty"`
}

type ContactJSON struct {
	ID      string `json:"id"`
	Address string `json:"address"`
}

type WSServer struct {
	node    *dht.Node
	routing *dht.RoutingTable
	logger  *log.Logger
	mu      sync.RWMutex
	clients map[net.Conn]struct{}
}

func NewWSServer(node *dht.Node, routing *dht.RoutingTable, logger *log.Logger) *WSServer {
	return &WSServer{
		node:    node,
		routing: routing,
		logger:  logger,
		clients: make(map[net.Conn]struct{}),
	}
}

func (s *WSServer) ListenAndServe(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWS)
	mux.HandleFunc("/health", s.handleHealth)
	s.logger.Printf("[ws] listening on %s", addr)
	return http.ListenAndServe(addr, mux)
}

func (s *WSServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"ok","node_id":"%s","address":"%s"}`, s.node.ID.String(), s.node.Address)
}

// handleWS performs the WebSocket handshake manually (no external deps).
func (s *WSServer) handleWS(w http.ResponseWriter, r *http.Request) {
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		http.Error(w, "expected websocket upgrade", http.StatusBadRequest)
		return
	}

	key := r.Header.Get("Sec-Websocket-Key")
	if key == "" {
		http.Error(w, "missing Sec-Websocket-Key", http.StatusBadRequest)
		return
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijack not supported", http.StatusInternalServerError)
		return
	}
	conn, rw, err := hj.Hijack()
	if err != nil {
		s.logger.Printf("[ws] hijack error: %v", err)
		return
	}

	accept := computeAccept(key)
	resp := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + accept + "\r\n\r\n"
	if _, err := conn.Write([]byte(resp)); err != nil {
		conn.Close()
		return
	}

	s.mu.Lock()
	s.clients[conn] = struct{}{}
	s.mu.Unlock()

	remote := conn.RemoteAddr().String()
	s.logger.Printf("[ws] client connected: %s", remote)

	go s.handleConn(conn, rw)
}

func (s *WSServer) handleConn(conn net.Conn, rw *bufio.ReadWriter) {
	defer func() {
		conn.Close()
		s.mu.Lock()
		delete(s.clients, conn)
		s.mu.Unlock()
		s.logger.Printf("[ws] client disconnected: %s", conn.RemoteAddr())
	}()

	for {
		frame, err := readFrame(rw.Reader)
		if err != nil {
			return
		}
		if len(frame) == 0 {
			continue
		}

		var msg Message
		if err := json.Unmarshal(frame, &msg); err != nil {
			s.logger.Printf("[ws] bad message from %s: %v", conn.RemoteAddr(), err)
			s.writeMessage(conn, Message{Type: MsgError, Error: "invalid JSON"})
			continue
		}

		s.logger.Printf("[ws] %s -> %s id=%s key=%s", conn.RemoteAddr(), msg.Type, msg.ID, msg.Key)
		response := s.dispatch(msg)
		s.writeMessage(conn, response)
	}
}

func (s *WSServer) dispatch(msg Message) Message {
	switch msg.Type {
	case MsgFindNode:
		return s.handleFindNode(msg)
	case MsgStore:
		return s.handleStore(msg)
	case MsgFindValue:
		return s.handleFindValue(msg)
	default:
		return Message{Type: MsgError, Error: fmt.Sprintf("unknown message type: %s", msg.Type)}
	}
}

func (s *WSServer) handleFindNode(msg Message) Message {
	if msg.ID == "" {
		return Message{Type: MsgError, Error: "missing id"}
	}
	target, err := dht.NodeIDFromString(msg.ID)
	if err != nil {
		// treat as arbitrary key, hash it
		target = dht.NodeIDFromBytes([]byte(msg.ID))
	}

	closest := s.routing.FindClosest(target, dht.K)
	nodes := make([]ContactJSON, len(closest))
	for i, c := range closest {
		nodes[i] = ContactJSON{ID: c.ID.String(), Address: c.Address}
	}

	s.logger.Printf("[ws] FIND_NODE %s -> %d results", msg.ID[:min(8, len(msg.ID))], len(nodes))
	return Message{Type: MsgResponse, Nodes: nodes}
}

func (s *WSServer) handleStore(msg Message) Message {
	if msg.Key == "" {
		return Message{Type: MsgError, Error: "missing key"}
	}
	ttl := time.Duration(msg.TTL) * time.Second
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	s.node.Store().Store(msg.Key, []byte(msg.Value), ttl)
	s.logger.Printf("[ws] STORE key=%s ttl=%v", msg.Key, ttl)
	return Message{Type: MsgResponse}
}

func (s *WSServer) handleFindValue(msg Message) Message {
	if msg.Key == "" {
		return Message{Type: MsgError, Error: "missing key"}
	}
	val, found := s.node.Store().FindValue(msg.Key)
	if found {
		s.logger.Printf("[ws] FIND_VALUE key=%s -> found", msg.Key)
		return Message{Type: MsgResponse, Key: msg.Key, Value: string(val)}
	}

	// Not found locally — return closest nodes.
	target := dht.NodeIDFromBytes([]byte(msg.Key))
	closest := s.routing.FindClosest(target, dht.K)
	nodes := make([]ContactJSON, len(closest))
	for i, c := range closest {
		nodes[i] = ContactJSON{ID: c.ID.String(), Address: c.Address}
	}
	s.logger.Printf("[ws] FIND_VALUE key=%s -> not found, returning %d nodes", msg.Key, len(nodes))
	return Message{Type: MsgResponse, Nodes: nodes}
}

func (s *WSServer) writeMessage(conn net.Conn, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	if err := writeFrame(conn, data); err != nil {
		s.logger.Printf("[ws] write error: %v", err)
	}
}

// -- WebSocket framing (RFC 6455) ------------------------------------------

func computeAccept(key string) string {
	const magic = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	h := sha1.New()
	h.Write([]byte(key + magic))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// readFrame reads a single WebSocket frame and returns the unmasked payload.
func readFrame(r *bufio.Reader) ([]byte, error) {
	// Byte 0: FIN + opcode
	b0, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	opcode := b0 & 0x0F
	if opcode == 8 { // close
		return nil, fmt.Errorf("close frame")
	}

	// Byte 1: MASK + payload length
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
	if _, err := readFull(r, payload); err != nil {
		return nil, err
	}

	if masked {
		for i := range payload {
			payload[i] ^= maskKey[i%4]
		}
	}
	return payload, nil
}

// writeFrame writes data as an unmasked text frame.
func writeFrame(conn net.Conn, data []byte) error {
	var header []byte
	header = append(header, 0x81) // FIN + text opcode
	n := len(data)
	if n <= 125 {
		header = append(header, byte(n))
	} else if n <= 65535 {
		header = append(header, 126, byte(n>>8), byte(n))
	} else {
		header = append(header, 127,
			0, 0, 0, 0,
			byte(n>>24), byte(n>>16), byte(n>>8), byte(n),
		)
	}
	out := append(header, data...)
	_, err := conn.Write(out)
	return err
}

func readFull(r *bufio.Reader, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		n, err := r.Read(buf[total:])
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
