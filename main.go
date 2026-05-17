package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ocp-node/dht"
	"ocp-node/server"
)

const (
	listenAddr = ":5000"
	nodeAddr   = "localhost:5000"
)

// Bootstrap nodes — update these with real peer addresses before deployment.
var bootstrapNodes = []string{
	// "peer1.ocp-network.example:5000",
	// "peer2.ocp-network.example:5000",
}

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds)

	logger.Println("[main] starting OCP DHT node")

	node, err := dht.NewNode(nodeAddr, logger)
	if err != nil {
		logger.Fatalf("[main] failed to create node: %v", err)
	}

	logger.Printf("[main] node ID: %s", node.ID.String())
	logger.Printf("[main] node address: %s", node.Address)

	routing := dht.NewRoutingTable(node, logger)

	// Bootstrap
	if len(bootstrapNodes) > 0 {
		logger.Printf("[main] bootstrapping with %d nodes", len(bootstrapNodes))
		node.Bootstrap(bootstrapNodes)
	} else {
		logger.Println("[main] no bootstrap nodes configured — running as seed node")
	}

	// Announce own address to the DHT.
	self := &dht.Contact{
		ID:       node.ID,
		Address:  node.Address,
		LastSeen: time.Now(),
	}
	routing.UpdateRoutingTable(self)
	logger.Printf("[main] announced self to routing table: %s", self)

	quit := make(chan struct{})

	// Background: bucket refresh.
	go routing.StartRefreshLoop(quit)

	// Background: store cleanup every 10 minutes.
	go node.Store().StartCleanupLoop(10*time.Minute, quit)

	// WebSocket server.
	ws := server.NewWSServer(node, routing, logger)

	go func() {
		logger.Printf("[main] WebSocket server starting on %s", listenAddr)
		if err := ws.ListenAndServe(listenAddr); err != nil {
			logger.Fatalf("[main] server error: %v", err)
		}
	}()

	// Graceful shutdown on SIGINT / SIGTERM.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig
	logger.Printf("[main] received signal %s, shutting down", s)
	close(quit)
	logger.Println("[main] shutdown complete")
}
