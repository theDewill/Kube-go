package kubenet

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	// Network configuration
	broadcastPort       = 9999
	statusPort          = 9998
	broadcastInterval   = 30 * time.Second
	nodeExpiration      = 5 * time.Minute
	persistenceInterval = 1 * time.Minute
	persistenceFile     = "node_registry.json"

	// Network protocol
	protocolVersion = 1

	// Node status constants
	StatusOnline  = "online"
	StatusLocked  = "locked"
	StatusOffline = "offline"
)

// Node represents a discovered machine in the network
type Node struct {
	ID          string    //`json:"id"`          // Unique identifier for the node
	IP          string    //`json:"ip"`          // IP address of the node
	Hostname    string    //`json:"hostname"`    // Hostname of the node
	LastSeen    time.Time //`json:"last_seen"`   // When this node was last seen
	Version     int       //`json:"version"`     // Protocol version
	Broadcasted bool      //`json:"broadcasted"` // Whether this is our own node
	Status      string    //`json:"status"`      // Node status (online, locked, offline)
}

// NodeRegistry maintains the state of discovered nodes
type NodeRegistry struct {
	sync.RWMutex
	nodes           map[string]*Node       // Key is node ID
	localNode       *Node                  // Our own node
	persistDir      string                 // Directory for persistence file
	statusListeners []func(string, string) // Callbacks for status changes (nodeID, status)
}

// BroadcastMessage is the structure sent over the network
type BroadcastMessage struct {
	Version  int    //`json:"version"`
	NodeID   string //`json:"node_id"`
	Hostname string //`json:"hostname"`
	Status   string //`json:"status"`
}

// StatusUpdateMessage is the structure sent for status updates
type StatusUpdateMessage struct {
	Version   int       //`json:"version"`
	NodeID    string    //`json:"node_id"`
	Status    string    //`json:"status"`
	Timestamp time.Time //`json:"timestamp"`
}

// NewNodeRegistry creates a new node registry
func NewNodeRegistry(persistDir string) (*NodeRegistry, error) {
	log.Println("Creating Node Registry..")
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get hostname: %w", err)
	}

	localIP, err := getLocalIP()
	if err != nil {
		return nil, fmt.Errorf("failed to get local IP: %w", err)
	}

	registry := &NodeRegistry{
		nodes:           make(map[string]*Node),
		persistDir:      persistDir,
		statusListeners: make([]func(string, string), 0),
		localNode: &Node{
			ID:          generateNodeID(hostname, localIP),
			IP:          localIP,
			Hostname:    hostname,
			LastSeen:    time.Now(),
			Version:     protocolVersion,
			Broadcasted: true,
			Status:      StatusOnline,
		},
	}

	// Add our own node to the registry
	registry.nodes[registry.localNode.ID] = registry.localNode

	return registry, nil
}

// GetNodesForFrontend returns the current nodes in a format suitable for the Wails frontend
func (nr *NodeRegistry) GetNodesForFrontend() []map[string]interface{} {
	nr.RLock()
	defer nr.RUnlock()

	nodes := make([]map[string]interface{}, 0, len(nr.nodes))
	for _, node := range nr.nodes {
		nodeMap := map[string]interface{}{
			"id":          node.ID,
			"ip":          node.IP,
			"hostname":    node.Hostname,
			"lastSeen":    node.LastSeen.Format(time.RFC3339),
			"version":     node.Version,
			"broadcasted": node.Broadcasted,
			"status":      node.Status,
			"isLocal":     node.ID == nr.localNode.ID,
		}
		nodes = append(nodes, nodeMap)
	}

	return nodes
}

// Start begins the broadcasting and listening processes
func (nr *NodeRegistry) Start(Done <-chan struct{}) error {
	// Try to load previous state
	if err := nr.loadFromDisk(); err != nil {
		log.Printf("Warning: could not load previous node state: %v", err)
	}

	// Start the broadcaster
	go nr.broadcastPresence(Done)

	// Start the listener
	go nr.listenForBroadcasts(Done)

	// Start the status update listener
	go nr.listenForStatusUpdates(Done)

	// Start the cleanup routine
	go nr.cleanupStaleNodes(Done)

	// Start persistence routine
	go nr.persistRegistryPeriodically(Done)

	return nil
}

// broadcastPresence periodically broadcasts our presence to the network
func (nr *NodeRegistry) broadcastPresence(Done <-chan struct{}) {
	ticker := time.NewTicker(broadcastInterval)
	defer ticker.Stop()

	conn, err := createBroadcastSocket()
	if err != nil {
		log.Printf("Failed to create broadcast socket: %v", err)
		return
	}
	defer conn.Close()

	for {
		select {
		case <-ticker.C:
			// Always get the latest status from our local node
			nr.RLock()
			message := BroadcastMessage{
				Version:  protocolVersion,
				NodeID:   nr.localNode.ID,
				Hostname: nr.localNode.Hostname,
				Status:   nr.localNode.Status,
			}
			nr.RUnlock()

			messageBytes, err := json.Marshal(message)
			if err != nil {
				log.Printf("Failed to marshal broadcast message: %v", err)
				continue
			}

			// Broadcast to all interfaces
			addrs, err := getBroadcastAddresses()
			if err != nil {
				log.Printf("Failed to get broadcast addresses: %v", err)
				continue
			}

			for _, addr := range addrs {
				udpAddr := &net.UDPAddr{
					IP:   addr,
					Port: broadcastPort,
				}

				if _, err := conn.WriteToUDP(messageBytes, udpAddr); err != nil {
					log.Printf("Failed to broadcast to %s: %v", addr, err)
				}
			}

		case <-Done:
			return
		}
	}
}

// listenForBroadcasts listens for incoming node broadcasts
func (nr *NodeRegistry) listenForBroadcasts(Done <-chan struct{}) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IPv4zero,
		Port: broadcastPort,
	})
	if err != nil {
		log.Printf("Failed to listen for broadcasts: %v", err)
		return
	}
	defer conn.Close()

	buffer := make([]byte, 1024)

	for {
		select {
		case <-Done:
			return
		default:
			// Set a deadline so we don't block forever
			if err := conn.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
				log.Printf("Failed to set read deadline: %v", err)
				continue
			}

			n, addr, err := conn.ReadFromUDP(buffer)
			if err != nil {
				if !errors.Is(err, os.ErrDeadlineExceeded) {
					log.Printf("Error reading from UDP: %v", err)
				}
				continue
			}

			// Skip our own broadcasts
			if addr.IP.String() == nr.localNode.IP {
				continue
			}

			var msg BroadcastMessage
			if err := json.Unmarshal(buffer[:n], &msg); err != nil {
				log.Printf("Failed to unmarshal broadcast message from %s: %v", addr, err)
				continue
			}

			// Validate protocol version
			if msg.Version != protocolVersion {
				log.Printf("Received message with incompatible version %d from %s", msg.Version, addr)
				continue
			}

			// Update the registry
			nr.Lock()
			node, exists := nr.nodes[msg.NodeID]
			if !exists {
				node = &Node{
					ID:       msg.NodeID,
					IP:       addr.IP.String(),
					Hostname: msg.Hostname,
					Version:  msg.Version,
					Status:   msg.Status,
				}
				nr.nodes[msg.NodeID] = node
				log.Printf("Discovered new node: %s (%s) with status %s", msg.Hostname, addr.IP, msg.Status)
			} else {
				// Check if status changed
				oldStatus := node.Status
				if oldStatus != msg.Status {
					log.Printf("Node %s (%s) status changed from %s to %s",
						msg.Hostname, addr.IP, oldStatus, msg.Status)

					// Notify listeners about status change
					for _, listener := range nr.statusListeners {
						go listener(msg.NodeID, msg.Status)
					}
				}
				node.Status = msg.Status
			}
			node.LastSeen = time.Now()
			nr.Unlock()
		}
	}
}

// SetNodeStatus updates the status of the local node and propagates the change
func (nr *NodeRegistry) SetNodeStatus(status string) error {
	nr.Lock()
	// Store old status to check if it changed
	oldStatus := nr.localNode.Status
	nr.localNode.Status = status
	nr.Unlock()

	// If status has changed, propagate it immediately
	if oldStatus != status {
		return nr.propagateStatusChange(status)
	}
	return nil
}

// propagateStatusChange sends a status update to all nodes
func (nr *NodeRegistry) propagateStatusChange(status string) error {
	nr.RLock()
	message := StatusUpdateMessage{
		Version:   protocolVersion,
		NodeID:    nr.localNode.ID,
		Status:    status,
		Timestamp: time.Now(),
	}
	nr.RUnlock()

	messageBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal status update message: %w", err)
	}

	// Create a UDP connection
	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IPv4zero,
		Port: 0,
	})
	if err != nil {
		return fmt.Errorf("failed to create status update socket: %w", err)
	}
	defer conn.Close()

	// Get all nodes except self
	nr.RLock()
	targets := make([]string, 0, len(nr.nodes)-1)
	for id, node := range nr.nodes {
		if id != nr.localNode.ID {
			targets = append(targets, node.IP)
		}
	}
	nr.RUnlock()

	// Send to all nodes
	for _, targetIP := range targets {
		udpAddr := &net.UDPAddr{
			IP:   net.ParseIP(targetIP),
			Port: statusPort,
		}

		if _, err := conn.WriteToUDP(messageBytes, udpAddr); err != nil {
			log.Printf("Failed to send status update to %s: %v", targetIP, err)
			// Continue to try other nodes
		}
	}

	return nil
}

// listenForStatusUpdates listens for incoming status update messages
func (nr *NodeRegistry) listenForStatusUpdates(Done <-chan struct{}) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IPv4zero,
		Port: statusPort,
	})
	if err != nil {
		log.Printf("Failed to listen for status updates: %v", err)
		return
	}
	defer conn.Close()

	buffer := make([]byte, 1024)

	for {
		select {
		case <-Done:
			return
		default:
			// Set a deadline so we don't block forever
			if err := conn.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
				log.Printf("Failed to set read deadline for status updates: %v", err)
				continue
			}

			n, addr, err := conn.ReadFromUDP(buffer)
			if err != nil {
				if !errors.Is(err, os.ErrDeadlineExceeded) {
					log.Printf("Error reading status update from UDP: %v", err)
				}
				continue
			}

			var msg StatusUpdateMessage
			if err := json.Unmarshal(buffer[:n], &msg); err != nil {
				log.Printf("Failed to unmarshal status update from %s: %v", addr, err)
				continue
			}

			// Validate protocol version
			if msg.Version != protocolVersion {
				log.Printf("Received status update with incompatible version %d from %s", msg.Version, addr)
				continue
			}

			// Update the node status in registry
			nr.Lock()
			node, exists := nr.nodes[msg.NodeID]
			if exists {
				oldStatus := node.Status
				node.Status = msg.Status
				node.LastSeen = time.Now() // Update last seen time

				log.Printf("Updated node %s status from %s to %s", node.Hostname, oldStatus, msg.Status)

				// Notify status listeners
				for _, listener := range nr.statusListeners {
					go listener(msg.NodeID, msg.Status)
				}
			} else {
				log.Printf("Received status update for unknown node ID: %s", msg.NodeID)
			}
			nr.Unlock()
		}
	}
}

// AddStatusListener registers a callback function to be called when a node's status changes
func (nr *NodeRegistry) AddStatusListener(callback func(string, string)) {
	nr.Lock()
	nr.statusListeners = append(nr.statusListeners, callback)
	nr.Unlock()
}

// cleanupStaleNodes periodically removes nodes that haven't been seen recently
func (nr *NodeRegistry) cleanupStaleNodes(Done <-chan struct{}) {
	ticker := time.NewTicker(nodeExpiration / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			nr.Lock()
			now := time.Now()
			for id, node := range nr.nodes {
				if !node.Broadcasted && now.Sub(node.LastSeen) > nodeExpiration {
					log.Printf("Removing stale node: %s (%s)", node.Hostname, node.IP)

					// Notify listeners of node going offline before removal
					for _, listener := range nr.statusListeners {
						go listener(id, StatusOffline)
					}

					delete(nr.nodes, id)
				}
			}
			nr.Unlock()

		case <-Done:
			return
		}
	}
}

// persistRegistryPeriodically saves the node registry to disk periodically
func (nr *NodeRegistry) persistRegistryPeriodically(Done <-chan struct{}) {
	ticker := time.NewTicker(persistenceInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := nr.saveToDisk(); err != nil {
				log.Printf("Failed to persist node registry: %v", err)
			}

		case <-Done:
			// Make a final attempt to save before exiting
			if err := nr.saveToDisk(); err != nil {
				log.Printf("Failed to persist node registry on shutdown: %v", err)
			}
			return
		}
	}
}

// saveToDisk saves the current node registry to disk
func (nr *NodeRegistry) saveToDisk() error {
	nr.RLock()
	defer nr.RUnlock()

	// Create a copy of the nodes without our own broadcasted node
	nodesToSave := make([]*Node, 0, len(nr.nodes))
	for _, node := range nr.nodes {
		if !node.Broadcasted {
			nodesToSave = append(nodesToSave, node)
		}
	}

	data, err := json.Marshal(nodesToSave)
	if err != nil {
		return fmt.Errorf("failed to marshal node data: %w", err)
	}

	filePath := filepath.Join(nr.persistDir, persistenceFile)
	tmpPath := filePath + ".tmp"

	// Write to temporary file first
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Atomically rename the temporary file
	if err := os.Rename(tmpPath, filePath); err != nil {
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	return nil
}

// loadFromDisk loads the node registry from disk
func (nr *NodeRegistry) loadFromDisk() error {
	filePath := filepath.Join(nr.persistDir, persistenceFile)
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No existing file is not an error
		}
		return fmt.Errorf("failed to read persistence file: %w", err)
	}

	var nodes []*Node
	if err := json.Unmarshal(data, &nodes); err != nil {
		return fmt.Errorf("failed to unmarshal node data: %w", err)
	}

	nr.Lock()
	defer nr.Unlock()
	for _, node := range nodes {
		// Mark loaded nodes as offline initially until we hear from them
		if node.Status == StatusOnline {
			node.Status = StatusOffline
		}
		nr.nodes[node.ID] = node
	}

	return nil
}

// GetNodes returns a copy of the current node list
func (nr *NodeRegistry) GetNodes() []Node {
	nr.RLock()
	defer nr.RUnlock()

	nodes := make([]Node, 0, len(nr.nodes))
	for _, node := range nr.nodes {
		nodeCopy := *node // Create a copy to avoid race conditions
		nodes = append(nodes, nodeCopy)
	}
	return nodes
}

// GetNodeByID returns a specific node by ID
func (nr *NodeRegistry) GetNodeByID(nodeID string) (Node, bool) {
	nr.RLock()
	defer nr.RUnlock()

	if node, exists := nr.nodes[nodeID]; exists {
		return *node, true
	}
	return Node{}, false
}

// Helper functions

func getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}

	return "", errors.New("no non-loopback IPv4 address found")
}

func getBroadcastAddresses() ([]net.IP, error) {
	var broadcastIPs []net.IP

	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range interfaces {
		// Skip interfaces that are down or don't support multicast
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagMulticast == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			if ipnet.IP.To4() == nil {
				continue // Skip IPv6
			}

			broadcast := calculateBroadcastAddress(ipnet)
			if broadcast != nil {
				broadcastIPs = append(broadcastIPs, broadcast)
			}
		}
	}

	if len(broadcastIPs) == 0 {
		return nil, errors.New("no broadcast addresses found")
	}

	return broadcastIPs, nil
}

func calculateBroadcastAddress(ipnet *net.IPNet) net.IP {
	ip := ipnet.IP.To4()
	if ip == nil {
		return nil
	}

	mask := ipnet.Mask
	if len(mask) != net.IPv4len {
		return nil
	}

	broadcast := make(net.IP, net.IPv4len)
	for i := 0; i < net.IPv4len; i++ {
		broadcast[i] = ip[i] | ^mask[i]
	}

	return broadcast
}

func createBroadcastSocket() (*net.UDPConn, error) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return nil, err
	}

	// Enable broadcast
	if err := conn.SetWriteBuffer(1024); err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}

func generateNodeID(hostname, ip string) string {
	return fmt.Sprintf("%s-%s", hostname, ip)
}

// LockNode locks the local node (typically when user is away)
func (nr *NodeRegistry) LockNode() error {
	return nr.SetNodeStatus(StatusLocked)
}

// UnlockNode unlocks the local node (when user returns)
func (nr *NodeRegistry) UnlockNode() error {
	return nr.SetNodeStatus(StatusOnline)
}

/*
func main() {
	// Example usage
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create persistence directory if it doesn't exist
	persistDir := "./data"
	if err := os.MkdirAll(persistDir, 0755); err != nil {
		log.Fatalf("Failed to create persistence directory: %v", err)
	}

	registry, err := NewNodeRegistry(persistDir)
	if err != nil {
		log.Fatalf("Failed to create node registry: %v", err)
	}

	// Register a status change listener as an example
	registry.AddStatusListener(func(nodeID, status string) {
		log.Printf("Status change notification: Node %s status changed to %s", nodeID, status)
	})

	if err := registry.Start(ctx); err != nil {
		log.Fatalf("Failed to start node registry: %v", err)
	}

	// Print discovered nodes periodically
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				nodes := registry.GetNodes()
				log.Printf("Discovered %d nodes:", len(nodes))
				for _, node := range nodes {
					log.Printf("- %s (%s) status: %s, last seen %v",
						node.Hostname, node.IP, node.Status, node.LastSeen)
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	// Simulate status change for demonstration (in a real app, this would be triggered by events)
	time.AfterFunc(2*time.Minute, func() {
		log.Println("Simulating node lock (user away)...")
		if err := registry.LockNode(); err != nil {
			log.Printf("Failed to lock node: %v", err)
		}
	})

	time.AfterFunc(4*time.Minute, func() {
		log.Println("Simulating node unlock (user returned)...")
		if err := registry.UnlockNode(); err != nil {
			log.Printf("Failed to unlock node: %v", err)
		}
	})

	// Wait for termination signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Shutting down...")

	// Set node as offline before shutting down
	if err := registry.SetNodeStatus(StatusOffline); err != nil {
		log.Printf("Failed to set node status to offline: %v", err)
	}

	cancel()
	time.Sleep(1 * time.Second) // Give time for cleanup
}
*/
