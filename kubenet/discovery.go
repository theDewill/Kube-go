package kubenet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

const (
	// Network configuration
	broadcastPort       = 9999
	broadcastInterval   = 30 * time.Second
	nodeExpiration      = 5 * time.Minute
	persistenceInterval = 1 * time.Minute
	persistenceFile     = "node_registry.json"

	// Network protocol
	protocolVersion = 1
)

// Node represents a discovered machine in the network
type Node struct {
	ID          string    `json:"id"`          // Unique identifier for the node
	IP          string    `json:"ip"`          // IP address of the node
	Hostname    string    `json:"hostname"`    // Hostname of the node
	LastSeen    time.Time `json:"last_seen"`   // When this node was last seen
	Version     int       `json:"version"`     // Protocol version
	Broadcasted bool      `json:"broadcasted"` // Whether this is our own node
}

// NodeRegistry maintains the state of discovered nodes
type NodeRegistry struct {
	sync.RWMutex
	nodes      map[string]*Node // Key is node ID
	localNode  *Node            // Our own node
	persistDir string           // Directory for persistence file
}

// BroadcastMessage is the structure sent over the network
type BroadcastMessage struct {
	Version  int    `json:"version"`
	NodeID   string `json:"node_id"`
	Hostname string `json:"hostname"`
}

// NewNodeRegistry creates a new node registry
func NewNodeRegistry(persistDir string) (*NodeRegistry, error) {
	println("Creating Node Registry..")
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get hostname: %w", err)
	}

	localIP, err := getLocalIP()
	if err != nil {
		return nil, fmt.Errorf("failed to get local IP: %w", err)
	}

	registry := &NodeRegistry{
		nodes:      make(map[string]*Node),
		persistDir: persistDir,
		localNode: &Node{
			ID:          generateNodeID(hostname, localIP),
			IP:          localIP,
			Hostname:    hostname,
			LastSeen:    time.Now(),
			Version:     protocolVersion,
			Broadcasted: true,
		},
	}

	// Add our own node to the registry
	registry.nodes[registry.localNode.ID] = registry.localNode

	return registry, nil
}

// Start begins the broadcasting and listening processes
func (nr *NodeRegistry) Start(ctx context.Context) error {
	// Try to load previous state
	if err := nr.loadFromDisk(); err != nil {
		log.Printf("Warning: could not load previous node state: %v", err)
	}

	// Start the broadcaster
	go nr.broadcastPresence(ctx)

	// Start the listener
	go nr.listenForBroadcasts(ctx)

	// Start the cleanup routine
	go nr.cleanupStaleNodes(ctx)

	// Start persistence routine
	go nr.persistRegistryPeriodically(ctx)

	return nil
}

// broadcastPresence periodically broadcasts our presence to the network
func (nr *NodeRegistry) broadcastPresence(ctx context.Context) {
	ticker := time.NewTicker(broadcastInterval)
	defer ticker.Stop()

	conn, err := createBroadcastSocket()
	if err != nil {
		log.Printf("Failed to create broadcast socket: %v", err)
		return
	}
	defer conn.Close()

	message := BroadcastMessage{
		Version:  protocolVersion,
		NodeID:   nr.localNode.ID,
		Hostname: nr.localNode.Hostname,
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal broadcast message: %v", err)
		return
	}

	for {
		select {
		case <-ticker.C:
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

		case <-ctx.Done():
			return
		}
	}
}

// listenForBroadcasts listens for incoming node broadcasts
func (nr *NodeRegistry) listenForBroadcasts(ctx context.Context) {
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
		case <-ctx.Done():
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
				}
				nr.nodes[msg.NodeID] = node
				log.Printf("Discovered new node: %s (%s)", msg.Hostname, addr.IP)
			}
			node.LastSeen = time.Now()
			nr.Unlock()
		}
	}
}

// cleanupStaleNodes periodically removes nodes that haven't been seen recently
func (nr *NodeRegistry) cleanupStaleNodes(ctx context.Context) {
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
					delete(nr.nodes, id)
				}
			}
			nr.Unlock()

		case <-ctx.Done():
			return
		}
	}
}

// persistRegistryPeriodically saves the node registry to disk periodically
func (nr *NodeRegistry) persistRegistryPeriodically(ctx context.Context) {
	ticker := time.NewTicker(persistenceInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := nr.saveToDisk(); err != nil {
				log.Printf("Failed to persist node registry: %v", err)
			}

		case <-ctx.Done():
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
		nodes = append(nodes, *node)
	}
	return nodes
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
					log.Printf("- %s (%s) last seen %v", node.Hostname, node.IP, node.LastSeen)
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for termination signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Shutting down...")
	cancel()
	time.Sleep(1 * time.Second) // Give time for cleanup
}
