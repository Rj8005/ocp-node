package dht

import (
	"crypto/rand"
	"log"
	"time"
)

const bucketRefreshInterval = 15 * time.Minute

type RoutingTable struct {
	node   *Node
	logger *log.Logger
}

func NewRoutingTable(node *Node, logger *log.Logger) *RoutingTable {
	return &RoutingTable{node: node, logger: logger}
}

func (rt *RoutingTable) FindClosest(target NodeID, k int) []*Contact {
	return rt.node.FindClosest(target, k)
}

func (rt *RoutingTable) UpdateRoutingTable(contact *Contact) {
	contact.LastSeen = time.Now()
	rt.node.AddContact(contact)
}

// Refresh queries a random node ID in each stale bucket.
func (rt *RoutingTable) Refresh() {
	now := time.Now()
	for i := 0; i < BucketLen; i++ {
		b := rt.node.buckets[i]
		b.mu.RLock()
		stale := b.lastSeen.IsZero() || now.Sub(b.lastSeen) > bucketRefreshInterval
		b.mu.RUnlock()

		if !stale {
			continue
		}

		randomID := rt.randomIDInBucket(i)
		rt.logger.Printf("[routing] refreshing bucket %d with random ID %s", i, randomID.String()[:8])
		closest := rt.node.FindClosest(randomID, K)
		rt.logger.Printf("[routing] bucket %d refresh found %d contacts", i, len(closest))
	}
}

// randomIDInBucket returns a random NodeID that falls into bucket index i.
// Bucket i contains nodes whose XOR distance with our ID has prefix length i.
func (rt *RoutingTable) randomIDInBucket(i int) NodeID {
	var id NodeID
	_, _ = rand.Read(id[:])

	// Copy the first i bits from our node ID then flip bit i.
	byteIdx := i / 8
	bitIdx := uint(7 - (i % 8))

	for b := 0; b < byteIdx; b++ {
		id[b] = rt.node.ID[b]
	}

	// Set byte at byteIdx: copy high bits from our ID, flip bit i, randomise lower bits.
	mask := byte(0xFF) << (bitIdx + 1)
	id[byteIdx] = (rt.node.ID[byteIdx] & mask) | (^rt.node.ID[byteIdx] & (1 << bitIdx)) | (id[byteIdx] & ^(mask | (1 << bitIdx)))

	return id
}

// StartRefreshLoop runs periodic bucket refresh in the background.
func (rt *RoutingTable) StartRefreshLoop(quit <-chan struct{}) {
	ticker := time.NewTicker(bucketRefreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			rt.Refresh()
		case <-quit:
			return
		}
	}
}
