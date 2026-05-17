package dht

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"sort"
	"sync"
	"time"
)

const (
	K         = 20
	Alpha     = 3
	IDLen     = 20
	BucketLen = 160
)

type NodeID [IDLen]byte

func NewNodeID() NodeID {
	var id NodeID
	_, err := rand.Read(id[:])
	if err != nil {
		panic(fmt.Sprintf("failed to generate node ID: %v", err))
	}
	return id
}

func NodeIDFromString(s string) (NodeID, error) {
	var id NodeID
	b, err := hex.DecodeString(s)
	if err != nil {
		return id, err
	}
	if len(b) != IDLen {
		return id, fmt.Errorf("invalid node ID length: %d", len(b))
	}
	copy(id[:], b)
	return id, nil
}

func NodeIDFromBytes(b []byte) NodeID {
	return NodeID(sha1.Sum(b))
}

func (id NodeID) String() string {
	return hex.EncodeToString(id[:])
}

func (id NodeID) XOR(other NodeID) NodeID {
	var result NodeID
	for i := 0; i < IDLen; i++ {
		result[i] = id[i] ^ other[i]
	}
	return result
}

func (id NodeID) PrefixLen() int {
	for i, b := range id {
		if b == 0 {
			continue
		}
		for j := 7; j >= 0; j-- {
			if b>>uint(j)&1 != 0 {
				return i*8 + (7 - j)
			}
		}
	}
	return BucketLen
}

func (id NodeID) Less(other NodeID) bool {
	for i := 0; i < IDLen; i++ {
		if id[i] < other[i] {
			return true
		}
		if id[i] > other[i] {
			return false
		}
	}
	return false
}

type Contact struct {
	ID      NodeID
	Address string
	LastSeen time.Time
}

func (c Contact) String() string {
	return fmt.Sprintf("%s@%s", c.ID.String()[:8], c.Address)
}

type bucket struct {
	mu       sync.RWMutex
	contacts []*Contact
	lastSeen time.Time
}

func (b *bucket) addOrUpdate(c *Contact) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i, existing := range b.contacts {
		if existing.ID == c.ID {
			b.contacts[i].Address = c.Address
			b.contacts[i].LastSeen = time.Now()
			b.lastSeen = time.Now()
			return true
		}
	}

	if len(b.contacts) < K {
		b.contacts = append(b.contacts, c)
		b.lastSeen = time.Now()
		return true
	}
	return false
}

func (b *bucket) remove(id NodeID) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i, c := range b.contacts {
		if c.ID == id {
			b.contacts = append(b.contacts[:i], b.contacts[i+1:]...)
			return
		}
	}
}

func (b *bucket) getContacts() []*Contact {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]*Contact, len(b.contacts))
	copy(out, b.contacts)
	return out
}

type Node struct {
	ID       NodeID
	Address  string
	buckets  [BucketLen]*bucket
	store    *Store
	mu       sync.RWMutex
	conn     *net.UDPConn
	quit     chan struct{}
	logger   *log.Logger
}

func NewNode(address string, logger *log.Logger) (*Node, error) {
	n := &Node{
		ID:      NewNodeID(),
		Address: address,
		store:   NewStore(),
		quit:    make(chan struct{}),
		logger:  logger,
	}
	for i := 0; i < BucketLen; i++ {
		n.buckets[i] = &bucket{}
	}
	return n, nil
}

func (n *Node) bucketIndex(id NodeID) int {
	dist := n.ID.XOR(id)
	idx := dist.PrefixLen()
	if idx >= BucketLen {
		idx = BucketLen - 1
	}
	return idx
}

func (n *Node) AddContact(c *Contact) {
	if c.ID == n.ID {
		return
	}
	idx := n.bucketIndex(c.ID)
	if !n.buckets[idx].addOrUpdate(c) {
		n.logger.Printf("[DHT] bucket %d full, dropping contact %s", idx, c)
	} else {
		n.logger.Printf("[DHT] added contact %s to bucket %d", c, idx)
	}
}

func (n *Node) FindClosest(target NodeID, count int) []*Contact {
	type distContact struct {
		dist    NodeID
		contact *Contact
	}
	var all []distContact

	for i := 0; i < BucketLen; i++ {
		for _, c := range n.buckets[i].getContacts() {
			all = append(all, distContact{
				dist:    target.XOR(c.ID),
				contact: c,
			})
		}
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].dist.Less(all[j].dist)
	})

	if count > len(all) {
		count = len(all)
	}
	result := make([]*Contact, count)
	for i := 0; i < count; i++ {
		result[i] = all[i].contact
	}
	return result
}

func (n *Node) Bootstrap(bootstrapAddrs []string) {
	for _, addr := range bootstrapAddrs {
		c := &Contact{
			ID:       NodeIDFromBytes([]byte(addr)),
			Address:  addr,
			LastSeen: time.Now(),
		}
		n.AddContact(c)
		n.logger.Printf("[DHT] bootstrap contact added: %s", c)
	}
	n.logger.Printf("[DHT] bootstrap complete with %d nodes", len(bootstrapAddrs))
}

func (n *Node) Store() *Store {
	return n.store
}
