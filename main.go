package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Rj8005/ocp-node/dht"
	"github.com/Rj8005/ocp-node/server"
)

const (
	httpPort = 5000
	nodeAddr = "localhost:5000"
)

// Bootstrap nodes — update these with real peer addresses before deployment.
var bootstrapNodes = []string{
	// "ocp-bootstrap-1.onrender.com:443",
	// "ocp-bootstrap-2.onrender.com:443",
}

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

	if len(bootstrapNodes) > 0 {
		logger.Printf("[main] bootstrapping with %d nodes", len(bootstrapNodes))
		node.Bootstrap(bootstrapNodes)
	} else {
		logger.Println("[main] no bootstrap nodes configured — running as seed node")
	}

	// Announce own address to the DHT.
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
