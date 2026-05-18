package server

import (
	"encoding/binary"
	"log"
	"net"
	"sync"
	"time"
)

const (
	turnPort        = 3478
	allocationTTL   = 10 * time.Minute
	stunBindingReq  = 0x0001
	stunBindingResp = 0x0101
	turnAllocReq    = 0x0003
	turnAllocResp   = 0x0103
	magicCookie     = 0x2112A442
)

type TURNAllocation struct {
	clientAddr *net.UDPAddr
	relayAddr  *net.UDPAddr
	relayConn  *net.UDPConn
	peerAddr   *net.UDPAddr
	lastSeen   time.Time
	expiry     time.Time
}

type TURNServer struct {
	conn        *net.UDPConn
	allocations sync.Map
	mu          sync.Mutex
}

func NewTURNServer() *TURNServer {
	return &TURNServer{}
}

func (t *TURNServer) Start() error {
	addr, err := net.ResolveUDPAddr("udp", ":3478")
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	t.conn = conn
	log.Printf("[turn] STUN/TURN server listening on UDP :3478")

	go t.cleanup()
	go t.serve()
	return nil
}

func (t *TURNServer) serve() {
	buf := make([]byte, 65536)
	for {
		n, addr, err := t.conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("[turn] read error: %v", err)
			continue
		}
		go t.handlePacket(buf[:n], addr)
	}
}

func (t *TURNServer) handlePacket(data []byte, addr *net.UDPAddr) {
	if len(data) < 20 {
		return
	}

	msgType := binary.BigEndian.Uint16(data[0:2])
	magic := binary.BigEndian.Uint32(data[4:8])

	if magic != magicCookie {
		return
	}

	switch msgType {
	case stunBindingReq:
		t.handleSTUNBinding(data, addr)
	case turnAllocReq:
		t.handleTURNAlloc(data, addr)
	default:
		t.handleRelay(data, addr)
	}
}

func (t *TURNServer) handleSTUNBinding(data []byte, addr *net.UDPAddr) {
	txID := data[8:20]
	resp := make([]byte, 32)
	binary.BigEndian.PutUint16(resp[0:2], stunBindingResp)
	binary.BigEndian.PutUint16(resp[2:4], 12)
	binary.BigEndian.PutUint32(resp[4:8], magicCookie)
	copy(resp[8:20], txID)
	// XOR-MAPPED-ADDRESS attribute
	binary.BigEndian.PutUint16(resp[20:22], 0x0020)
	binary.BigEndian.PutUint16(resp[22:24], 8)
	resp[24] = 0
	resp[25] = 0x01 // IPv4
	port := uint16(addr.Port) ^ uint16(magicCookie>>16)
	binary.BigEndian.PutUint16(resp[26:28], port)
	ip := addr.IP.To4()
	mc := uint32(magicCookie)
	for i := 0; i < 4; i++ {
		resp[28+i] = ip[i] ^ byte(mc>>(24-uint(i)*8))
	}
	t.conn.WriteToUDP(resp, addr)
	log.Printf("[turn] STUN binding response to %s", addr)
}

func (t *TURNServer) handleTURNAlloc(data []byte, addr *net.UDPAddr) {
	relayConn, err := net.ListenUDP("udp", &net.UDPAddr{Port: 0})
	if err != nil {
		log.Printf("[turn] relay allocation failed: %v", err)
		return
	}

	relayAddr := relayConn.LocalAddr().(*net.UDPAddr)
	alloc := &TURNAllocation{
		clientAddr: addr,
		relayAddr:  relayAddr,
		relayConn:  relayConn,
		lastSeen:   time.Now(),
		expiry:     time.Now().Add(allocationTTL),
	}

	t.allocations.Store(addr.String(), alloc)

	txID := data[8:20]
	resp := make([]byte, 56)
	binary.BigEndian.PutUint16(resp[0:2], turnAllocResp)
	binary.BigEndian.PutUint16(resp[2:4], 36)
	binary.BigEndian.PutUint32(resp[4:8], magicCookie)
	copy(resp[8:20], txID)

	t.conn.WriteToUDP(resp, addr)
	log.Printf("[turn] allocated relay port %d for %s", relayAddr.Port, addr)

	go t.relayData(alloc)
}

func (t *TURNServer) handleRelay(data []byte, addr *net.UDPAddr) {
	val, ok := t.allocations.Load(addr.String())
	if !ok {
		return
	}
	alloc := val.(*TURNAllocation)
	alloc.lastSeen = time.Now()
	if alloc.peerAddr != nil {
		alloc.relayConn.WriteToUDP(data, alloc.peerAddr)
	}
}

func (t *TURNServer) relayData(alloc *TURNAllocation) {
	buf := make([]byte, 65536)
	for {
		alloc.relayConn.SetReadDeadline(time.Now().Add(allocationTTL))
		n, peerAddr, err := alloc.relayConn.ReadFromUDP(buf)
		if err != nil {
			break
		}
		alloc.peerAddr = peerAddr
		t.conn.WriteToUDP(buf[:n], alloc.clientAddr)
	}
	t.allocations.Delete(alloc.clientAddr.String())
	alloc.relayConn.Close()
	log.Printf("[turn] allocation expired for %s", alloc.clientAddr)
}

func (t *TURNServer) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		t.allocations.Range(func(key, val interface{}) bool {
			alloc := val.(*TURNAllocation)
			if now.After(alloc.expiry) {
				t.allocations.Delete(key)
				alloc.relayConn.Close()
			}
			return true
		})
	}
}
