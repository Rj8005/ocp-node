package main

import (
	"bufio"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Rj8005/ocp-node/dht"
	"github.com/Rj8005/ocp-node/server"
)

const (
	httpPort = 5000
	nodeAddr = "localhost:5000"
)

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds)
	logger.Println("[main] starting OCP DHT node")

	node, err := dht.NewNode(nodeAddr, logger)
	if err != nil {
		logger.Fatalf("[main] failed to create node: %v", err)
	}

	logger.Printf("[main] node ID: %s", node.IDHex())
	logger.Printf("[main] node address: %s", node.Address())

	routing := dht.NewRoutingTable(node, logger)

	// Announce self to routing table.
	self := &dht.Contact{
		ID:       node.ID,
		Address:  node.Address(),
		LastSeen: time.Now(),
	}
	routing.UpdateRoutingTable(self)
	logger.Printf("[main] announced self to routing table: %s", self)

	quit := make(chan struct{})
	go routing.StartRefreshLoop(quit)
	go node.Store().StartCleanupLoop(10*time.Minute, quit)

	// Connect to bootstrap peers from environment.
	bootstrapPeers := os.Getenv("BOOTSTRAP_PEERS")
	if bootstrapPeers != "" {
		peers := strings.Split(bootstrapPeers, ",")
		logger.Printf("[main] connecting to %d bootstrap peers", len(peers))
		for _, peer := range peers {
			peer = strings.TrimSpace(peer)
			if peer == "" {
				continue
			}
			go connectToPeer(node, peer)
		}
	} else {
		logger.Println("[main] no bootstrap peers configured — running as seed node")
	}

	httpServer := server.NewHTTPServer(node, httpPort)
	go func() {
		logger.Printf("[main] HTTP server starting on port %d", httpPort)
		if err := httpServer.Start(); err != nil {
			logger.Fatalf("[main] server error: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig
	logger.Printf("[main] received signal %s, shutting down", s)
	close(quit)
	logger.Println("[main] shutdown complete")
}

// connectToPeer dials wsURL and retries indefinitely on failure.
func connectToPeer(node *dht.Node, wsURL string) {
	for {
		err := tryConnect(node, wsURL)
		if err != nil {
			log.Printf("[peer] failed to connect to %s: %v — retrying in 30s", wsURL, err)
			time.Sleep(30 * time.Second)
		}
	}
}

// tryConnect opens a WebSocket connection to wsURL, registers the peer, sends
// a FIND_NODE announcement, then reads messages until the connection drops.
func tryConnect(node *dht.Node, wsURL string) error {
	log.Printf("[peer] connecting to %s", wsURL)

	u, err := url.Parse(wsURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	host := u.Host
	path := u.Path
	if path == "" {
		path = "/"
	}
	if u.RawQuery != "" {
		path += "?" + u.RawQuery
	}

	// Dial raw TCP (plain or TLS).
	var rawConn net.Conn
	switch u.Scheme {
	case "wss":
		if !strings.Contains(host, ":") {
			host += ":443"
		}
		rawConn, err = tls.Dial("tcp", host, &tls.Config{ServerName: u.Hostname()})
	case "ws":
		if !strings.Contains(host, ":") {
			host += ":80"
		}
		rawConn, err = net.Dial("tcp", host)
	default:
		return fmt.Errorf("unsupported scheme %q (use ws:// or wss://)", u.Scheme)
	}
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer rawConn.Close()

	// Send the HTTP/1.1 upgrade request.
	var keyBytes [16]byte
	rand.Read(keyBytes[:])
	wsKey := base64.StdEncoding.EncodeToString(keyBytes[:])

	upgradeReq := "GET " + path + " HTTP/1.1\r\n" +
		"Host: " + u.Host + "\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: " + wsKey + "\r\n" +
		"Sec-WebSocket-Version: 13\r\n\r\n"
	if _, err := rawConn.Write([]byte(upgradeReq)); err != nil {
		return fmt.Errorf("upgrade request: %w", err)
	}

	// Read the 101 Switching Protocols response.
	reader := bufio.NewReader(rawConn)
	resp, err := http.ReadResponse(reader, nil)
	if err != nil {
		return fmt.Errorf("upgrade response: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSwitchingProtocols {
		return fmt.Errorf("expected 101 Switching Protocols, got %d", resp.StatusCode)
	}

	node.AddPeer(wsURL)
	defer node.RemovePeer(wsURL)
	log.Printf("[peer] connected to %s", wsURL)

	// Announce ourselves with FIND_NODE so the peer adds us to its routing table.
	ping := fmt.Sprintf(`{"type":"FIND_NODE","id":"%s"}`, node.IDHex())
	if err := writeClientFrame(rawConn, []byte(ping)); err != nil {
		return fmt.Errorf("FIND_NODE send: %w", err)
	}

	// Read messages until the peer closes or the read deadline fires.
	for {
		rawConn.SetReadDeadline(time.Now().Add(90 * time.Second))
		msg, err := readClientFrame(reader)
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}
		log.Printf("[peer] message from %s: %s", wsURL, string(msg))
	}
}

// writeClientFrame writes data as a masked WebSocket text frame.
// RFC 6455 §5.3: all frames sent from client to server MUST be masked.
func writeClientFrame(conn net.Conn, data []byte) error {
	var maskKey [4]byte
	if _, err := rand.Read(maskKey[:]); err != nil {
		return err
	}
	n := len(data)
	header := []byte{0x81} // FIN + text opcode
	switch {
	case n <= 125:
		header = append(header, byte(n)|0x80)
	case n <= 65535:
		header = append(header, 0x80|126, byte(n>>8), byte(n))
	default:
		header = append(header, 0x80|127, 0, 0, 0, 0,
			byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
	}
	header = append(header, maskKey[:]...)
	payload := make([]byte, n)
	for i, b := range data {
		payload[i] = b ^ maskKey[i%4]
	}
	_, err := conn.Write(append(header, payload...))
	return err
}

// readClientFrame reads a single unmasked WebSocket frame from the server.
func readClientFrame(r *bufio.Reader) ([]byte, error) {
	b0, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	if b0&0x0F == 8 {
		return nil, fmt.Errorf("close frame received")
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
	return payload, nil
}
