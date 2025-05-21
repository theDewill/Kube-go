package kubenet

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
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
	fileChunkPort = 9997
	maxChunkSize  = 1024 * 1024 // 1MB chunks
)

type ChunkInfo struct {
	ChunkID      string `json:"chunk_id"`       // Unique identifier for the chunk
	FileID       string `json:"file_id"`        // ID of the original file
	FileName     string `json:"file_name"`      // Original file name
	ChunkIndex   int    `json:"chunk_index"`    // Index of this chunk in the file
	TotalChunks  int    `json:"total_chunks"`   // Total number of chunks for this file
	ChunkSize    int    `json:"chunk_size"`     // Size of this chunk in bytes
	OriginalSize int64  `json:"original_size"`  // Original file size
	IsLocal      bool   `json:"is_local"`       // Whether this chunk belongs to the local node
	OwnerNodeID  string `json:"owner_node_id"`  // ID of the node that owns the original file
	StoredNodeID string `json:"stored_node_id"` // ID of the node storing this chunk
	Encrypted    bool   `json:"encrypted"`      // Whether the chunk is encrypted
}

// ChunkTransferMessage is used to transfer chunks between nodes
type ChunkTransferMessage struct {
	ChunkInfo ChunkInfo `json:"chunk_info"` // Metadata about the chunk
	Data      []byte    `json:"data"`       // The actual chunk data
}

// ChunkRequestMessage is used to request a chunk from another node
type ChunkRequestMessage struct {
	ChunkID string `json:"chunk_id"` // ID of the requested chunk
	NodeID  string `json:"node_id"`  // ID of the requesting node
}

// InitializeChunkDatabase initializes the SQLite database for chunk management
func (nr *NodeRegistry) InitializeChunkDatabase(dbPath string) error {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Create tables for chunk tracking
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS chunks (
			chunk_id TEXT PRIMARY KEY,
			file_id TEXT NOT NULL,
			file_name TEXT NOT NULL,
			chunk_index INTEGER NOT NULL,
			total_chunks INTEGER NOT NULL,
			chunk_size INTEGER NOT NULL,
			original_size INTEGER NOT NULL,
			is_local BOOLEAN NOT NULL,
			owner_node_id TEXT NOT NULL,
			stored_node_id TEXT NOT NULL,
			encrypted BOOLEAN NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_chunks_file_id ON chunks(file_id);
		CREATE INDEX IF NOT EXISTS idx_chunks_owner_node_id ON chunks(owner_node_id);
	`)
	if err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	return nil
}

// StartChunkTransferService starts a service to listen for chunk transfer requests
func (nr *NodeRegistry) StartChunkTransferService(ctx context.Context, kuberestsDir string, dbPath string) error {
	// Ensure directory exists
	if err := os.MkdirAll(kuberestsDir, 0755); err != nil {
		return fmt.Errorf("failed to create kuberests directory: %w", err)
	}

	// Start listening for chunk transfers
	listener, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IPv4zero,
		Port: fileChunkPort,
	})
	if err != nil {
		return fmt.Errorf("failed to listen for chunk transfers: %w", err)
	}

	log.Printf("Started chunk transfer service on port %d", fileChunkPort)

	go func() {
		// Close the listener when the goroutine exits
		defer listener.Close()

		// Set up database connection inside the goroutine
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			log.Printf("Failed to open database in chunk transfer service: %v", err)
			return
		}
		defer db.Close()

		buffer := make([]byte, maxChunkSize+1024) // Extra space for metadata

		for {
			select {
			case <-ctx.Done():
				log.Println("Chunk transfer service shutting down")
				return
			default:
				// Set a read deadline to avoid blocking forever
				if err := listener.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
					// Check if the context is done before logging error
					select {
					case <-ctx.Done():
						return
					default:
						log.Printf("Failed to set read deadline: %v", err)
						continue
					}
				}

				n, addr, err := listener.ReadFromUDP(buffer)
				if err != nil {
					// Check if the context is done before handling error
					select {
					case <-ctx.Done():
						return
					default:
						if !errors.Is(err, os.ErrDeadlineExceeded) {
							log.Printf("Error reading chunk data: %v", err)
						}
						continue
					}
				}

				// Process the received data in a separate goroutine
				go func(data []byte, size int, remoteAddr *net.UDPAddr) {
					// Create a fresh database connection for this handler
					handlerDB, err := sql.Open("sqlite3", dbPath)
					if err != nil {
						log.Printf("Failed to open database for chunk handler: %v", err)
						return
					}
					defer handlerDB.Close()

					// Try to unmarshal as a ChunkTransferMessage
					var transferMsg ChunkTransferMessage
					if err := json.Unmarshal(data[:size], &transferMsg); err == nil {
						// Process chunk transfer
						nr.handleChunkTransfer(transferMsg, kuberestsDir, handlerDB, remoteAddr)
						return
					}

					// Try to unmarshal as a ChunkRequestMessage
					var requestMsg ChunkRequestMessage
					if err := json.Unmarshal(data[:size], &requestMsg); err == nil {
						// Process chunk request
						nr.handleChunkRequest(requestMsg, kuberestsDir, listener, remoteAddr)
						return
					}

					log.Printf("Received unknown message type from %s", remoteAddr)
				}(buffer[:n], n, addr)
			}
		}
	}()

	return nil
}

// handleChunkTransfer processes a received chunk and stores it
func (nr *NodeRegistry) handleChunkTransfer(msg ChunkTransferMessage, kuberestsDir string, db *sql.DB, remoteAddr *net.UDPAddr) {
	log.Printf("Received chunk %s for file %s from %s", msg.ChunkInfo.ChunkID, msg.ChunkInfo.FileID, remoteAddr)

	// Store the chunk in the kuberests directory
	chunkPath := filepath.Join(kuberestsDir, msg.ChunkInfo.ChunkID)
	if err := ioutil.WriteFile(chunkPath, msg.Data, 0644); err != nil {
		log.Printf("Failed to write chunk to disk: %v", err)
		return
	}

	// Store chunk information in the database
	_, err := db.Exec(`
		INSERT OR REPLACE INTO chunks
		(chunk_id, file_id, file_name, chunk_index, total_chunks, chunk_size, original_size, is_local, owner_node_id, stored_node_id, encrypted)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		msg.ChunkInfo.ChunkID,
		msg.ChunkInfo.FileID,
		msg.ChunkInfo.FileName,
		msg.ChunkInfo.ChunkIndex,
		msg.ChunkInfo.TotalChunks,
		msg.ChunkInfo.ChunkSize,
		msg.ChunkInfo.OriginalSize,
		false, // Not a local chunk
		msg.ChunkInfo.OwnerNodeID,
		nr.localNode.ID, // Store locally
		msg.ChunkInfo.Encrypted)

	if err != nil {
		log.Printf("Failed to save chunk info to database: %v", err)
		return
	}

	log.Printf("Successfully stored chunk %s for file %s", msg.ChunkInfo.ChunkID, msg.ChunkInfo.FileID)
}

// handleChunkRequest processes a request for a chunk and sends it back
func (nr *NodeRegistry) handleChunkRequest(msg ChunkRequestMessage, kuberestsDir string, conn *net.UDPConn, remoteAddr *net.UDPAddr) {
	log.Printf("Received request for chunk %s from node %s", msg.ChunkID, msg.NodeID)

	// Open the database
	db, err := sql.Open("sqlite3", filepath.Join(kuberestsDir, "../fileidx.sqlite"))
	if err != nil {
		log.Printf("Failed to open database: %v", err)
		return
	}
	defer db.Close()

	// Query for the chunk information
	var chunkInfo ChunkInfo
	err = db.QueryRow(`
		SELECT chunk_id, file_id, file_name, chunk_index, total_chunks, chunk_size, original_size, is_local, owner_node_id, stored_node_id, encrypted
		FROM chunks WHERE chunk_id = ?
	`, msg.ChunkID).Scan(
		&chunkInfo.ChunkID,
		&chunkInfo.FileID,
		&chunkInfo.FileName,
		&chunkInfo.ChunkIndex,
		&chunkInfo.TotalChunks,
		&chunkInfo.ChunkSize,
		&chunkInfo.OriginalSize,
		&chunkInfo.IsLocal,
		&chunkInfo.OwnerNodeID,
		&chunkInfo.StoredNodeID,
		&chunkInfo.Encrypted,
	)
	if err != nil {
		log.Printf("Failed to find chunk %s in database: %v", msg.ChunkID, err)
		return
	}

	// Read the chunk data
	chunkPath := filepath.Join(kuberestsDir, chunkInfo.ChunkID)
	chunkData, err := ioutil.ReadFile(chunkPath)
	if err != nil {
		log.Printf("Failed to read chunk file: %v", err)
		return
	}

	// Create transfer message
	transferMsg := ChunkTransferMessage{
		ChunkInfo: chunkInfo,
		Data:      chunkData,
	}

	// Serialize the message
	msgData, err := json.Marshal(transferMsg)
	if err != nil {
		log.Printf("Failed to marshal chunk transfer message: %v", err)
		return
	}

	// Send the chunk back
	_, err = conn.WriteToUDP(msgData, remoteAddr)
	if err != nil {
		log.Printf("Failed to send chunk: %v", err)
		return
	}

	log.Printf("Successfully sent chunk %s to node %s", msg.ChunkID, msg.NodeID)
}

// DistributeFileChunks distributes file chunks to other nodes in the network
func (nr *NodeRegistry) DistributeFileChunks(fileID string, filePath string, fileName string, chunkDir string, dbPath string) error {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file size
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}
	fileSize := fileInfo.Size()

	// Calculate number of chunks
	numChunks := int((fileSize + maxChunkSize - 1) / maxChunkSize)
	if numChunks < 1 {
		numChunks = 1
	}

	// Get available nodes for distribution
	nr.RLock()
	availableNodes := make([]*Node, 0, len(nr.nodes))
	for id, node := range nr.nodes {
		if id != nr.localNode.ID && node.Status == StatusOnline {
			availableNodes = append(availableNodes, node)
		}
	}
	nr.RUnlock()

	// If no other nodes are available, store all chunks locally
	if len(availableNodes) == 0 {
		log.Println("No other nodes available, storing all chunks locally")
	}

	// Create a buffer for reading chunks
	buffer := make([]byte, maxChunkSize)

	// Open database connection
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Begin a transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Will be committed if no error

	// Prepare chunk insertion statement
	stmt, err := tx.Prepare(`
		INSERT INTO chunks
		(chunk_id, file_id, file_name, chunk_index, total_chunks, chunk_size, original_size, is_local, owner_node_id, stored_node_id, encrypted)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Process each chunk
	for i := 0; i < numChunks; i++ {
		// Read a chunk from the file
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read chunk: %w", err)
		}
		if n == 0 {
			break // End of file
		}

		// Generate a chunk ID using SHA-256 of fileID + index
		chunkIDSource := fmt.Sprintf("%s-%d", fileID, i)
		chunkIDHash := sha256.Sum256([]byte(chunkIDSource))
		chunkID := base64.URLEncoding.EncodeToString(chunkIDHash[:])

		// Encrypt the chunk
		encryptedData, err := encryptData(buffer[:n])
		if err != nil {
			return fmt.Errorf("failed to encrypt chunk: %w", err)
		}

		// Decide where to store this chunk
		var targetNode *Node
		isLocal := true
		storedNodeID := nr.localNode.ID

		if len(availableNodes) > 0 {
			// Select a node based on chunk index for even distribution
			targetNode = availableNodes[i%len(availableNodes)]
			isLocal = false
			storedNodeID = targetNode.ID
		}

		// Create chunk info
		chunkInfo := ChunkInfo{
			ChunkID:      chunkID,
			FileID:       fileID,
			FileName:     fileName,
			ChunkIndex:   i,
			TotalChunks:  numChunks,
			ChunkSize:    n,
			OriginalSize: fileSize,
			IsLocal:      isLocal,
			OwnerNodeID:  nr.localNode.ID,
			StoredNodeID: storedNodeID,
			Encrypted:    true,
		}

		// Store chunk info in the database
		_, err = stmt.Exec(
			chunkInfo.ChunkID,
			chunkInfo.FileID,
			chunkInfo.FileName,
			chunkInfo.ChunkIndex,
			chunkInfo.TotalChunks,
			chunkInfo.ChunkSize,
			chunkInfo.OriginalSize,
			chunkInfo.IsLocal,
			chunkInfo.OwnerNodeID,
			chunkInfo.StoredNodeID,
			chunkInfo.Encrypted,
		)
		if err != nil {
			return fmt.Errorf("failed to insert chunk info: %w", err)
		}

		if isLocal {
			// Store the chunk locally
			chunkPath := filepath.Join(chunkDir, chunkID)
			if err := ioutil.WriteFile(chunkPath, encryptedData, 0644); err != nil {
				return fmt.Errorf("failed to write local chunk: %w", err)
			}
		} else {
			// Send the chunk to the target node
			if err := nr.sendChunkToNode(chunkInfo, encryptedData, targetNode); err != nil {
				// If we fail to send to remote node, store locally as a fallback
				log.Printf("Failed to send chunk to node %s: %v, storing locally", targetNode.ID, err)

				// Update the database to mark as local
				_, err = tx.Exec(
					"UPDATE chunks SET is_local = ?, stored_node_id = ? WHERE chunk_id = ?",
					true, nr.localNode.ID, chunkID,
				)
				if err != nil {
					return fmt.Errorf("failed to update chunk info: %w", err)
				}

				// Store locally
				chunkPath := filepath.Join(chunkDir, chunkID)
				if err := ioutil.WriteFile(chunkPath, encryptedData, 0644); err != nil {
					return fmt.Errorf("failed to write fallback local chunk: %w", err)
				}
			}
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("Successfully distributed file %s in %d chunks", fileID, numChunks)
	return nil
}

// ReassembleFile reconstructs a file from its chunks
func (nr *NodeRegistry) ReassembleFile(fileID string, outputPath string, kuberestsDir string, dbPath string) error {
	// Open database connection
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Get file information
	var fileName string
	var totalChunks int
	var originalSize int64

	err = db.QueryRow(
		"SELECT file_name, total_chunks, original_size FROM chunks WHERE file_id = ? LIMIT 1",
		fileID,
	).Scan(&fileName, &totalChunks, &originalSize)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Create the output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	// Get information about all chunks for this file
	rows, err := db.Query(
		"SELECT chunk_id, chunk_index, is_local, stored_node_id FROM chunks WHERE file_id = ? ORDER BY chunk_index",
		fileID,
	)
	if err != nil {
		return fmt.Errorf("failed to query chunks: %w", err)
	}
	defer rows.Close()

	// Prepare a map of chunks to collect
	type chunkMetadata struct {
		ChunkID      string
		ChunkIndex   int
		IsLocal      bool
		StoredNodeID string
	}

	chunkMap := make(map[int]chunkMetadata)
	for rows.Next() {
		var cm chunkMetadata
		if err := rows.Scan(&cm.ChunkID, &cm.ChunkIndex, &cm.IsLocal, &cm.StoredNodeID); err != nil {
			return fmt.Errorf("failed to scan chunk row: %w", err)
		}
		chunkMap[cm.ChunkIndex] = cm
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating chunk rows: %w", err)
	}

	// Check if we have information about all chunks
	if len(chunkMap) != totalChunks {
		return fmt.Errorf("missing chunks: found %d of %d", len(chunkMap), totalChunks)
	}

	// Create a wait group for parallel fetch operations
	var wg sync.WaitGroup
	chunkDataMap := make(map[int][]byte, totalChunks)
	chunkMapMutex := sync.Mutex{}
	var fetchError error
	var errorMutex sync.Mutex

	// Process each chunk
	for i := 0; i < totalChunks; i++ {
		cm, exists := chunkMap[i]
		if !exists {
			return fmt.Errorf("missing chunk index %d", i)
		}

		wg.Add(1)
		go func(index int, metadata chunkMetadata) {
			defer wg.Done()

			var chunkData []byte
			var err error

			if metadata.IsLocal {
				// Read the chunk locally
				chunkPath := filepath.Join(kuberestsDir, metadata.ChunkID)
				chunkData, err = ioutil.ReadFile(chunkPath)
				if err != nil {
					errorMutex.Lock()
					fetchError = fmt.Errorf("failed to read local chunk %d: %w", index, err)
					errorMutex.Unlock()
					return
				}
			} else {
				// Request the chunk from the remote node
				chunkData, err = nr.requestChunkFromNode(metadata.ChunkID, metadata.StoredNodeID)
				if err != nil {
					errorMutex.Lock()
					fetchError = fmt.Errorf("failed to request chunk %d from node %s: %w",
						index, metadata.StoredNodeID, err)
					errorMutex.Unlock()
					return
				}
			}

			// Decrypt the chunk
			decryptedData, err := decryptData(chunkData)
			if err != nil {
				errorMutex.Lock()
				fetchError = fmt.Errorf("failed to decrypt chunk %d: %w", index, err)
				errorMutex.Unlock()
				return
			}

			// Store the decrypted chunk data
			chunkMapMutex.Lock()
			chunkDataMap[index] = decryptedData
			chunkMapMutex.Unlock()
		}(i, cm)
	}

	// Wait for all chunks to be processed
	wg.Wait()

	// Check if there was an error during fetching
	if fetchError != nil {
		return fetchError
	}

	// Write chunks to the output file in order
	for i := 0; i < totalChunks; i++ {
		data, exists := chunkDataMap[i]
		if !exists {
			return fmt.Errorf("chunk %d missing from result map", i)
		}

		if _, err := outputFile.Write(data); err != nil {
			return fmt.Errorf("failed to write chunk %d to output file: %w", i, err)
		}
	}

	// Ensure the file is the correct size
	if err := outputFile.Truncate(originalSize); err != nil {
		return fmt.Errorf("failed to truncate output file: %w", err)
	}

	log.Printf("Successfully reassembled file %s from %d chunks", fileName, totalChunks)
	return nil
}

// sendChunkToNode sends a chunk to another node
func (nr *NodeRegistry) sendChunkToNode(chunkInfo ChunkInfo, chunkData []byte, targetNode *Node) error {
	// Create a UDP connection
	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{
		IP:   net.ParseIP(targetNode.IP),
		Port: fileChunkPort,
	})
	if err != nil {
		return fmt.Errorf("failed to dial UDP: %w", err)
	}
	defer conn.Close()

	// Create the transfer message
	transferMsg := ChunkTransferMessage{
		ChunkInfo: chunkInfo,
		Data:      chunkData,
	}

	// Serialize the message
	msgData, err := json.Marshal(transferMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal chunk transfer message: %w", err)
	}

	// Send the message
	_, err = conn.Write(msgData)
	if err != nil {
		return fmt.Errorf("failed to send chunk: %w", err)
	}

	log.Printf("Sent chunk %s to node %s", chunkInfo.ChunkID, targetNode.ID)
	return nil
}

// requestChunkFromNode requests a chunk from another node
func (nr *NodeRegistry) requestChunkFromNode(chunkID string, nodeID string) ([]byte, error) {
	// Find the node
	nr.RLock()
	node, exists := nr.nodes[nodeID]
	nr.RUnlock()
	if !exists {
		return nil, fmt.Errorf("node %s not found", nodeID)
	}

	// Create a UDP connection
	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{
		IP:   net.ParseIP(node.IP),
		Port: fileChunkPort,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to dial UDP: %w", err)
	}
	defer conn.Close()

	// Create the request message
	requestMsg := ChunkRequestMessage{
		ChunkID: chunkID,
		NodeID:  nr.localNode.ID,
	}

	// Serialize the message
	msgData, err := json.Marshal(requestMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal chunk request message: %w", err)
	}

	// Send the request
	_, err = conn.Write(msgData)
	if err != nil {
		return nil, fmt.Errorf("failed to send chunk request: %w", err)
	}

	// Wait for the response (with timeout)
	responseBuffer := make([]byte, maxChunkSize+1024) // Extra space for metadata
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, _, err := conn.ReadFromUDP(responseBuffer)
	if err != nil {
		return nil, fmt.Errorf("failed to receive chunk: %w", err)
	}

	// Parse the response
	var transferMsg ChunkTransferMessage
	if err := json.Unmarshal(responseBuffer[:n], &transferMsg); err != nil {
		return nil, fmt.Errorf("failed to parse chunk response: %w", err)
	}

	// Verify the chunk ID
	if transferMsg.ChunkInfo.ChunkID != chunkID {
		return nil, fmt.Errorf("received wrong chunk: expected %s, got %s",
			chunkID, transferMsg.ChunkInfo.ChunkID)
	}

	log.Printf("Received chunk %s from node %s", chunkID, nodeID)
	return transferMsg.Data, nil
}

// Encryption and decryption helpers

// encryptData encrypts the given data using AES-256-GCM
func encryptData(data []byte) ([]byte, error) {

	key := make([]byte, 32) // AES-256 key
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and prepend the key and nonce
	ciphertext := gcm.Seal(nil, nonce, data, nil)
	result := make([]byte, len(key)+len(nonce)+len(ciphertext))

	copy(result, key)
	copy(result[len(key):], nonce)
	copy(result[len(key)+len(nonce):], ciphertext)

	return result, nil
}

// decryptData decrypts the given data
func decryptData(data []byte) ([]byte, error) {
	// Extract the key, nonce, and ciphertext
	if len(data) < 32+12 { // Key + nonce
		return nil, fmt.Errorf("data too short to contain key and nonce")
	}

	key := data[:32]
	nonce := data[32 : 32+12] // GCM nonce is 12 bytes
	ciphertext := data[32+12:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

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
	ctx             context.Context
	cancel          context.CancelFunc
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
	ctx, cancel := context.WithCancel(context.Background())

	registry := &NodeRegistry{
		ctx:             ctx,
		cancel:          cancel,
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
func (nr *NodeRegistry) GetNodesForFrontend_old() []map[string]interface{} {
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

// GetNodesForFrontend returns the current nodes in a format suitable for the Wails frontend
func (nr *NodeRegistry) GetNodesForFrontend() []map[string]interface{} {
	nr.RLock()
	defer nr.RUnlock()

	// Get current time to calculate node status age
	now := time.Now()

	nodes := make([]map[string]interface{}, 0, len(nr.nodes))
	for _, node := range nr.nodes {
		// Calculate time since last seen for frontend display
		lastSeenDuration := now.Sub(node.LastSeen)
		lastSeenMinutes := int(lastSeenDuration.Minutes())

		var lastSeenText string
		if lastSeenMinutes < 1 {
			lastSeenText = "just now"
		} else if lastSeenMinutes == 1 {
			lastSeenText = "1 minute ago"
		} else if lastSeenMinutes < 60 {
			lastSeenText = fmt.Sprintf("%d minutes ago", lastSeenMinutes)
		} else {
			lastSeenHours := lastSeenMinutes / 60
			if lastSeenHours == 1 {
				lastSeenText = "1 hour ago"
			} else {
				lastSeenText = fmt.Sprintf("%d hours ago", lastSeenHours)
			}
		}

		// Determine if the node is connected to this node
		isConnected := node.Status == StatusOnline || node.Status == StatusLocked

		// Create frontend-friendly node object
		nodeMap := map[string]interface{}{
			"id":            node.ID,
			"ip":            node.IP,
			"hostname":      node.Hostname,
			"lastSeen":      node.LastSeen.Format(time.RFC3339), // ISO format for precise timestamp
			"lastSeenText":  lastSeenText,                       // Human-readable format
			"version":       node.Version,
			"broadcasted":   node.Broadcasted,
			"status":        node.Status,
			"isLocal":       node.ID == nr.localNode.ID,
			"isConnected":   isConnected,
			"connectedTime": lastSeenDuration.String(),
		}
		nodes = append(nodes, nodeMap)
	}

	return nodes
}
func (nr *NodeRegistry) GetContext() context.Context {
	return nr.ctx
}

// Start begins the broadcasting and listening processes

// MAYBE: change the sole context in NodeRegistry to App CTX if needed by adding param here
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
