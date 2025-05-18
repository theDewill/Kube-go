package kfiles

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"kube-go/kubenet"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
)

type StorageInfo struct {
	TotalUsed      int64   `json:"total_used"`      // Total bytes used
	TotalUsedMB    float64 `json:"total_used_mb"`   // Total MB used
	TotalUsedGB    float64 `json:"total_used_gb"`   // Total GB used
	PercentageUsed float64 `json:"percentage_used"` // Percentage of 10GB used
	TotalCapacity  int64   `json:"total_capacity"`  // Total capacity (10GB in bytes)
	RemainingBytes int64   `json:"remaining_bytes"` // Remaining bytes available
	RemainingGB    float64 `json:"remaining_gb"`    // Remaining GB available
}

// FileBrowser manages file operations within the Kube application
type FileBrowser struct {
	PlatformPath string                // Base platform-specific path
	KubeLoadsDir string                // Dedicated kubeloads directory
	KubeRestsDir string                // Directory for storing chunks from other nodes
	DbPath       string                // Path to the SQLite database
	NodeRegistry *kubenet.NodeRegistry // Reference to the NodeRegistry
}

// FileType represents a file or folder in the file system
type FileType struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Type             string    `json:"type"`
	Extension        string    `json:"extension,omitempty"`
	Path             string    `json:"path"`
	Size             string    `json:"size,omitempty"`
	SizeInBytes      int64     `json:"sizeInBytes,omitempty"`
	LastModified     string    `json:"lastModified"`
	LastModifiedDate time.Time `json:"lastModifiedDate"`
	IsRefrigerated   bool      `json:"isRefrigerated,omitempty"`
	CompressionRatio float64   `json:"compressionRatio,omitempty"`
	Owner            string    `json:"owner"`
	IsShared         bool      `json:"isShared"`
	SharedWith       []string  `json:"sharedWith,omitempty"`
	ItemCount        int       `json:"itemCount,omitempty"`
	IsDistributed    bool      `json:"isDistributed,omitempty"`
}

// DIAGNOSTICS
func (fb *FileBrowser) GetStorageUsage() (StorageInfo, error) {
	const totalCapacityGB = 10
	const totalCapacityBytes = totalCapacityGB * 1024 * 1024 * 1024 // 10GB in bytes

	var totalUsed int64

	// Walk through the kubeloads directory and calculate total size
	err := filepath.Walk(fb.KubeLoadsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Log the error but don't stop the walk
			log.Printf("Warning: Error accessing %s: %v", path, err)
			return nil
		}

		// Only count regular files, not directories
		if !info.IsDir() {
			totalUsed += info.Size()
		}

		return nil
	})

	if err != nil {
		return StorageInfo{}, fmt.Errorf("failed to calculate storage usage: %w", err)
	}

	// Calculate percentage
	percentageUsed := (float64(totalUsed) / float64(totalCapacityBytes)) * 100

	// Calculate remaining space
	remainingBytes := totalCapacityBytes - totalUsed
	if remainingBytes < 0 {
		remainingBytes = 0
	}

	// Create storage info
	storageInfo := StorageInfo{
		TotalUsed:      totalUsed,
		TotalUsedMB:    float64(totalUsed) / (1024 * 1024),
		TotalUsedGB:    float64(totalUsed) / (1024 * 1024 * 1024),
		PercentageUsed: percentageUsed,
		TotalCapacity:  totalCapacityBytes,
		RemainingBytes: remainingBytes,
		RemainingGB:    float64(remainingBytes) / (1024 * 1024 * 1024),
	}

	return storageInfo, nil
}

func (fb *FileBrowser) CheckStorageQuota(newFileSize int64) (bool, error) {
	storageInfo, err := fb.GetStorageUsage()
	if err != nil {
		return false, err
	}

	// Check if adding the new file would exceed the 10GB limit
	newTotal := storageInfo.TotalUsed + newFileSize
	return newTotal <= storageInfo.TotalCapacity, nil
}

// InitializeSettings creates a default settings.json file in the app's home directory
func InitializeSettings() error {
	// Get the platform-specific app directory
	homeDir, err := GetPlatformSpecificPath()
	if err != nil {
		return fmt.Errorf("failed to get app directory: %w", err)
	}

	// Ensure the directory exists
	if err := os.MkdirAll(homeDir, 0755); err != nil {
		return fmt.Errorf("failed to create app directory: %w", err)
	}

	jsonPath := filepath.Join(homeDir, "settings.json")

	// Check if settings.json already exists
	if _, err := os.Stat(jsonPath); err == nil {
		log.Printf("Settings file already exists at: %s", jsonPath)
		return nil // File already exists, no need to create
	}

	// Create default settings
	defaultSettings := AppSettings{
		GeminiAPIKey: "", // Empty - user needs to fill this
		OllamaURL:    "http://localhost:11434",
		OllamaModel:  "phi3:mini",
	}

	// Marshal to JSON with indentation for readability
	jsonData, err := json.MarshalIndent(defaultSettings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings to JSON: %w", err)
	}

	// Write to file
	if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	log.Printf("Default settings file created at: %s", jsonPath)
	log.Printf("Please edit the file to add your Gemini API key and customize other settings")

	return nil
}

// LaunchFileBrowser creates a new FileBrowser instance with the platform-specific path
func LaunchFileBrowser(nodeRegistry *kubenet.NodeRegistry) (*FileBrowser, error) {
	platformPath, err := GetPlatformSpecificPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get platform-specific path: %w", err)
	}

	if err := InitializeSettings(); err != nil {
		log.Printf("Warning: Failed to initialize settings: %v", err)
	}

	// Create the kubeloads directory
	kubeLoadsDir := filepath.Join(platformPath, "kubeloads")
	if err := os.MkdirAll(kubeLoadsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create kubeloads directory: %w", err)
	}
	kubeRestsDir := filepath.Join(platformPath, "kuberests")
	if err := os.MkdirAll(kubeRestsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create kuberests directory: %w", err)
	}
	dbPath := filepath.Join(platformPath, "fileidx.sqlite")

	fb := &FileBrowser{
		PlatformPath: platformPath,
		KubeLoadsDir: kubeLoadsDir,
		KubeRestsDir: kubeRestsDir,
		DbPath:       dbPath,
		NodeRegistry: nodeRegistry,
	}
	print("FileBrowser Created")
	if err := fb.initializeDatabase(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize the NodeRegistry's chunk transfer services
	if err := nodeRegistry.InitializeChunkDatabase(dbPath); err != nil {
		return nil, fmt.Errorf("failed to initialize chunk database: %w", err)
	}

	// Start the chunk transfer service - CHECKHERE FOR CTX
	ctx := nodeRegistry.GetContext() // Assuming you add a GetContext() method to NodeRegistry
	if err := nodeRegistry.StartChunkTransferService(ctx, kubeRestsDir, dbPath); err != nil {
		return nil, fmt.Errorf("failed to start chunk transfer service: %w", err)
	}

	return fb, nil
}

// initializeDatabase sets up the SQLite database for file management
func (fb *FileBrowser) initializeDatabase() error {
	db, err := sql.Open("sqlite3", fb.DbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Create tables for file management
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS files (
			file_id TEXT PRIMARY KEY,
			file_name TEXT NOT NULL,
			file_path TEXT NOT NULL,
			file_size INTEGER NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			is_distributed BOOLEAN NOT NULL DEFAULT 0,
			total_chunks INTEGER NOT NULL DEFAULT 1
		);

		CREATE INDEX IF NOT EXISTS idx_files_path ON files(file_path);
		CREATE INDEX IF NOT EXISTS idx_files_distributed ON files(is_distributed);
	`)
	if err != nil {
		return fmt.Errorf("failed to create files table: %w", err)
	}

	return nil
}

// getPlatformSpecificPath returns the appropriate directory path based on the OS
func GetPlatformSpecificPath() (string, error) {
	var basePath string
	var err error

	switch runtime.GOOS {
	case "windows":
		// Use LOCALAPPDATA for Windows
		basePath = filepath.Join(os.Getenv("LOCALAPPDATA"), "Kube")
	case "darwin":
		// Use ~/Library/Application Support for macOS
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		basePath = filepath.Join(homeDir, "Library", "Application Support", "Kube")
	default: // Linux and other Unix-like systems
		// Use ~/.local/share for Linux
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		// Check for XDG_DATA_HOME environment variable
		xdgDataHome := os.Getenv("XDG_DATA_HOME")
		if xdgDataHome != "" {
			basePath = filepath.Join(xdgDataHome, "Kube")
		} else {
			basePath = filepath.Join(homeDir, ".local", "share", "Kube")
		}
	}

	// Create the base directory if it doesn't exist
	if err = os.MkdirAll(basePath, 0755); err != nil {
		return "", fmt.Errorf("failed to create base directory: %w", err)
	}

	return basePath, nil
}

// GetCurrentPath returns the absolute path for a relative path within kubeloads
func (fb *FileBrowser) GetCurrentPath(relativePath string) string {
	// Replace forward slashes for Windows compatibility
	relativePath = strings.ReplaceAll(relativePath, "/", string(os.PathSeparator))

	// If it's the root, return the kubeloads directory
	if relativePath == "" || relativePath == "/" {
		return fb.KubeLoadsDir
	}

	// Remove leading slash if present
	if strings.HasPrefix(relativePath, "/") {
		relativePath = relativePath[1:]
	}

	return filepath.Join(fb.KubeLoadsDir, relativePath)
}

// UploadFile saves an uploaded file and optionally distributes it across nodes
// func (fb *FileBrowser) UploadFile(directoryPath string, fileName string, fileData []byte, distribute bool) error {
// 	if !isValidName(fileName) {
// 		return errors.New("invalid file name")
// 	}

// 	// Create a unique file ID
// 	fileID := uuid.New().String()

// 	// Calculate file hash for additional integrity verification
// 	hasher := sha256.New()
// 	hasher.Write(fileData)
// 	//fileHash := hex.EncodeToString(hasher.Sum(nil))

// 	// Create the target path in kubeloads
// 	relativePath := strings.TrimPrefix(directoryPath, "/")
// 	targetDir := filepath.Join(fb.KubeLoadsDir, relativePath)

// 	// Ensure the directory exists
// 	if err := os.MkdirAll(targetDir, 0755); err != nil {
// 		return fmt.Errorf("failed to create directory: %w", err)
// 	}

// 	// Use the fileID as filename to avoid collisions and for security
// 	storageFileName := fmt.Sprintf("%s-%s", fileID, fileName)
// 	targetPath := filepath.Join(targetDir, storageFileName)

// 	// Store file in database
// 	db, err := sql.Open("sqlite3", fb.DbPath)
// 	if err != nil {
// 		return fmt.Errorf("failed to open database: %w", err)
// 	}
// 	defer db.Close()

// 	// Begin transaction
// 	tx, err := db.Begin()
// 	if err != nil {
// 		return fmt.Errorf("failed to begin transaction: %w", err)
// 	}
// 	defer tx.Rollback() // Will be committed on success

// 	print("INSERTING TO DB")
// 	// Insert file record
// 	now := time.Now()
// 	_, err = tx.Exec(
// 		"INSERT INTO files (file_id, file_name, file_path, file_size, created_at, updated_at, is_distributed) VALUES (?, ?, ?, ?, ?, ?, ?)",
// 		fileID, fileName, filepath.Join(relativePath, storageFileName), len(fileData), now, now, distribute,
// 	)
// 	if err != nil {
// 		return fmt.Errorf("failed to insert file record: %w", err)
// 	}

// 	// Write the file to disk
// 	if err := ioutil.WriteFile(targetPath, fileData, 0644); err != nil {
// 		return fmt.Errorf("failed to write file: %w", err)
// 	}

// 	// If distribution is requested, chunk and distribute the file
// 	if distribute {
// 		// Count expected chunks
// 		chunkSize := 1024 * 1024 // 1MB per chunk
// 		totalChunks := (len(fileData) + chunkSize - 1) / chunkSize

// 		// Update total_chunks in the database
// 		_, err = tx.Exec(
// 			"UPDATE files SET total_chunks = ? WHERE file_id = ?",
// 			totalChunks, fileID,
// 		)
// 		if err != nil {
// 			return fmt.Errorf("failed to update total chunks: %w", err)
// 		}

// 		// Commit the transaction before distributing chunks
// 		if err := tx.Commit(); err != nil {
// 			return fmt.Errorf("failed to commit transaction: %w", err)
// 		}

// 		// Distribute file chunks
// 		if err := fb.NodeRegistry.DistributeFileChunks(
// 			fileID,
// 			targetPath,
// 			fileName,
// 			fb.KubeRestsDir,
// 			fb.DbPath,
// 		); err != nil {
// 			// File is already saved, so don't fail completely
// 			log.Printf("Warning: Failed to distribute file %s: %v", fileID, err)
// 			return nil
// 		}

// 		log.Printf("File %s successfully distributed across nodes", fileID)
// 	} else {
// 		// Commit the transaction
// 		if err := tx.Commit(); err != nil {
// 			return fmt.Errorf("failed to commit transaction: %w", err)
// 		}
// 	}

//		return nil
//	}
//
// UploadFile saves an uploaded file and optionally distributes it across nodes
func (fb *FileBrowser) UploadFile(directoryPath string, fileName string, fileData []byte, distribute bool) error {
	return fb.UploadFileWithModel(directoryPath, fileName, fileData, distribute, "ollama")
}

// UploadFileWithModel saves an uploaded file with specified LLM model for description generation
func (fb *FileBrowser) UploadFileWithModel(directoryPath string, fileName string, fileData []byte, distribute bool, modelType string) error {
	if !isValidName(fileName) {
		return errors.New("invalid file name")
	}

	// Create a unique file ID
	fileID := uuid.New().String()

	// Calculate file hash for additional integrity verification
	hasher := sha256.New()
	hasher.Write(fileData)
	//fileHash := hex.EncodeToString(hasher.Sum(nil))

	// Create the target path in kubeloads
	relativePath := strings.TrimPrefix(directoryPath, "/")
	targetDir := filepath.Join(fb.KubeLoadsDir, relativePath)

	// Ensure the directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Use the fileID as filename to avoid collisions and for security
	storageFileName := fmt.Sprintf("%s-%s", fileID, fileName)
	targetPath := filepath.Join(targetDir, storageFileName)

	// Store file in database
	db, err := sql.Open("sqlite3", fb.DbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Will be committed on success

	// Insert file record
	now := time.Now()
	_, err = tx.Exec(
		"INSERT INTO files (file_id, file_name, file_path, file_size, created_at, updated_at, is_distributed) VALUES (?, ?, ?, ?, ?, ?, ?)",
		fileID, fileName, filepath.Join(relativePath, storageFileName), len(fileData), now, now, distribute,
	)
	if err != nil {
		return fmt.Errorf("failed to insert file record: %w", err)
	}

	// Write the file to disk
	if err := ioutil.WriteFile(targetPath, fileData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Generate file description asynchronously with specified model
	go func() {
		if err := fb.GenerateFileDescription(fileID, targetPath, fileName, modelType); err != nil {
			log.Printf("Warning: Failed to generate description for file %s using %s: %v", fileID, modelType, err)
		} else {
			log.Printf("Successfully generated description for file %s using %s", fileID, modelType)
		}
	}()

	// If distribution is requested, chunk and distribute the file
	if distribute {
		// Count expected chunks
		chunkSize := 1024 * 1024 // 1MB per chunk
		totalChunks := (len(fileData) + chunkSize - 1) / chunkSize

		// Update total_chunks in the database
		_, err = tx.Exec(
			"UPDATE files SET total_chunks = ? WHERE file_id = ?",
			totalChunks, fileID,
		)
		if err != nil {
			return fmt.Errorf("failed to update total chunks: %w", err)
		}

		// Commit the transaction before distributing chunks
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}

		// Distribute file chunks
		if err := fb.NodeRegistry.DistributeFileChunks(
			fileID,
			targetPath,
			fileName,
			fb.KubeRestsDir,
			fb.DbPath,
		); err != nil {
			// File is already saved, so don't fail completely
			log.Printf("Warning: Failed to distribute file %s: %v", fileID, err)
			return nil
		}

		log.Printf("File %s successfully distributed across nodes", fileID)
	} else {
		// Commit the transaction
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
	}

	return nil
}

// Update the ReadSettingsFile method:
func (fb *FileBrowser) ReadSettingsFile() ([]byte, error) {
	settingsPath := filepath.Join(fb.PlatformPath, "settings.json")

	// Check if file exists
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		// If settings file doesn't exist, create it with defaults
		if err := InitializeSettings(); err != nil {
			return nil, fmt.Errorf("failed to initialize settings: %w", err)
		}
	}

	// Read the settings file
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read settings file: %w", err)
	}

	return data, nil
}

// Update the WriteSettingsFile method to ensure it overwrites:
func (fb *FileBrowser) WriteSettingsFile(data []byte) error {
	settingsPath := filepath.Join(fb.PlatformPath, "settings.json")

	// Validate that it's valid JSON
	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("invalid JSON data: %w", err)
	}

	// Write the file (this will overwrite existing file)
	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	log.Printf("Settings file updated at: %s", settingsPath)
	return nil
}

// ListDirectory lists files and folders in the specified directory
func (fb *FileBrowser) ListDirectory(path string) ([]FileType, error) {
	dirPath := fb.GetCurrentPath(path)

	// Check if directory exists
	info, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("directory does not exist: %s", path)
		}
		return nil, fmt.Errorf("failed to access directory: %w", err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", path)
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	// Open database to get distributed status
	db, err := sql.Open("sqlite3", fb.DbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Prepare a map to store distributed status by filename
	distributedFiles := make(map[string]bool)

	// Query all files from this directory
	rows, err := db.Query("SELECT file_path, is_distributed FROM files")
	if err != nil {
		log.Printf("Warning: Failed to query distributed files: %v", err)
	} else {
		defer rows.Close()

		for rows.Next() {
			var filePath string
			var isDistributed bool
			if err := rows.Scan(&filePath, &isDistributed); err != nil {
				log.Printf("Warning: Failed to scan row: %v", err)
				continue
			}
			distributedFiles[filePath] = isDistributed
		}
	}

	files := make([]FileType, 0, len(entries))

	for _, entry := range entries {
		entryPath := filepath.Join(dirPath, entry.Name())
		info, err := os.Stat(entryPath)
		if err != nil {
			continue // Skip files with errors
		}

		// Calculate relative path from kubeloads directory
		relativePath, err := filepath.Rel(fb.KubeLoadsDir, entryPath)
		if err != nil {
			continue // Skip files with errors
		}

		// Convert path separators to forward slashes for consistent API
		relativePath = "/" + strings.ReplaceAll(relativePath, string(os.PathSeparator), "/")

		// Extract the original filename (remove fileID prefix if present)
		displayName := entry.Name()
		if strings.Contains(displayName, "-") && !entry.IsDir() {
			// Try to extract the original filename
			parts := strings.SplitN(displayName, "-", 2)
			if len(parts) == 2 && isValidUUID(parts[0]) {
				displayName = parts[1]
			}
		}

		// Create a FileType object
		file := FileType{
			ID:               uuid.New().String(),
			Name:             displayName,
			Path:             relativePath,
			LastModifiedDate: info.ModTime(),
			LastModified:     formatLastModified(info.ModTime()),
			Owner:            "You", // Default owner
			IsShared:         false,
		}

		if entry.IsDir() {
			file.Type = "folder"
			file.ItemCount = countItems(entryPath)
		} else {
			file.Type = "file"
			file.SizeInBytes = info.Size()
			file.Size = formatSize(info.Size())
			file.Extension = getFileExtension(displayName)
			file.IsRefrigerated = isRefrigerated(entryPath)

			// Check if this file is distributed
			file.IsDistributed = distributedFiles[relativePath]

			if file.IsRefrigerated {
				file.CompressionRatio = 0.5 // Placeholder - would need actual implementation
			}
		}

		files = append(files, file)
	}

	return files, nil
}

// DownloadFile downloads a file, reassembling it if it's distributed
func (fb *FileBrowser) DownloadFile(path string) ([]byte, error) {
	// First, check if the file exists directly
	filePath := fb.GetCurrentPath(path)

	// Get just the filename
	fileName := filepath.Base(path)

	// Check if it's a distributed file
	isDistributed, fileID, err := fb.isDistributedFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to check if file is distributed: %w", err)
	}

	if isDistributed {
		// Create a temporary file for reassembly
		tempFile := filepath.Join(os.TempDir(), fileName)

		// Reassemble the file
		if err := fb.NodeRegistry.ReassembleFile(fileID, tempFile, fb.KubeRestsDir, fb.DbPath); err != nil {
			return nil, fmt.Errorf("failed to reassemble file: %w", err)
		}

		// Read the reassembled file
		data, err := ioutil.ReadFile(tempFile)

		// Clean up temp file
		os.Remove(tempFile)

		if err != nil {
			return nil, fmt.Errorf("failed to read reassembled file: %w", err)
		}

		return data, nil
	}

	// It's not distributed, just read it directly
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}

// isDistributedFile checks if a file is distributed and returns its fileID
func (fb *FileBrowser) isDistributedFile(path string) (bool, string, error) {
	// Normalize path
	relativePath := strings.TrimPrefix(path, "/")
	relativePath = strings.ReplaceAll(relativePath, "/", string(os.PathSeparator))

	// Open database
	db, err := sql.Open("sqlite3", fb.DbPath)
	if err != nil {
		return false, "", fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Query for the file
	var isDistributed bool
	var fileID string

	// Try exact path first
	err = db.QueryRow(
		"SELECT is_distributed, file_id FROM files WHERE file_path = ?",
		relativePath,
	).Scan(&isDistributed, &fileID)

	if err == sql.ErrNoRows {
		// Try partial match (file might be in the path)
		rows, err := db.Query(
			"SELECT is_distributed, file_id, file_path FROM files",
		)
		if err != nil {
			return false, "", fmt.Errorf("failed to query files: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var id string
			var isDist bool
			var filePath string
			if err := rows.Scan(&isDist, &id, &filePath); err != nil {
				continue
			}

			// Check if this file path ends with our path
			if strings.HasSuffix(filePath, relativePath) || strings.HasSuffix(relativePath, filePath) {
				isDistributed = isDist
				fileID = id
				break
			}
		}

		if fileID == "" {
			return false, "", nil // File not found in database
		}
	} else if err != nil {
		return false, "", fmt.Errorf("failed to query file: %w", err)
	}

	return isDistributed, fileID, nil
}

// CreateFolder creates a new folder at the specified path
func (fb *FileBrowser) CreateFolder(path string, name string) error {
	if !isValidName(name) {
		return errors.New("invalid folder name")
	}

	folderPath := filepath.Join(fb.GetCurrentPath(path), name)
	if _, err := os.Stat(folderPath); err == nil {
		return fmt.Errorf("folder already exists: %s", name)
	}

	return os.MkdirAll(folderPath, 0755)
}

// RenameItem renames a file or folder
func (fb *FileBrowser) RenameItem(path string, newName string) error {
	if !isValidName(newName) {
		return errors.New("invalid name")
	}

	itemPath := fb.GetCurrentPath(path)
	dir := filepath.Dir(itemPath)
	newPath := filepath.Join(dir, newName)

	if _, err := os.Stat(newPath); err == nil {
		return fmt.Errorf("an item with the name %s already exists", newName)
	}

	// Check if this is a distributed file
	isDistributed, fileID, err := fb.isDistributedFile(path)
	if err != nil {
		return fmt.Errorf("failed to check if file is distributed: %w", err)
	}

	// If it's distributed, update the database
	if isDistributed {
		db, err := sql.Open("sqlite3", fb.DbPath)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer db.Close()

		// Update the file_name in the database
		_, err = db.Exec(
			"UPDATE files SET file_name = ? WHERE file_id = ?",
			newName, fileID,
		)
		if err != nil {
			return fmt.Errorf("failed to update file name in database: %w", err)
		}
	}

	// Rename the file on disk
	return os.Rename(itemPath, newPath)
}

// DeleteItem deletes a file or folder
func (fb *FileBrowser) DeleteItem(path string) error {
	itemPath := fb.GetCurrentPath(path)

	info, err := os.Stat(itemPath)
	if err != nil {
		return fmt.Errorf("failed to access item: %w", err)
	}

	// Check if this is a distributed file
	isDistributed, fileID, err := fb.isDistributedFile(path)
	if err != nil {
		return fmt.Errorf("failed to check if file is distributed: %w", err)
	}

	// If it's distributed, delete the chunks and database entries
	if isDistributed {
		db, err := sql.Open("sqlite3", fb.DbPath)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer db.Close()

		// Delete all chunks associated with this file
		// Note: This doesn't delete chunks on other nodes, they'll be cleaned up by expiration
		_, err = db.Exec(
			"DELETE FROM chunks WHERE file_id = ?",
			fileID,
		)
		if err != nil {
			log.Printf("Warning: Failed to delete chunk records: %v", err)
		}

		// Delete the file entry
		_, err = db.Exec(
			"DELETE FROM files WHERE file_id = ?",
			fileID,
		)
		if err != nil {
			log.Printf("Warning: Failed to delete file record: %v", err)
		}
	}

	// Delete the file or directory
	if info.IsDir() {
		return os.RemoveAll(itemPath)
	}

	return os.Remove(itemPath)
}

// Helper functions

// isValidName checks if a file or folder name is valid
func isValidName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}

	// Check for invalid characters based on OS
	switch runtime.GOOS {
	case "windows":
		// Windows has more restrictions
		invalidChars := []string{"<", ">", ":", "\"", "/", "\\", "|", "?", "*"}
		for _, char := range invalidChars {
			if strings.Contains(name, char) {
				return false
			}
		}
	default:
		// Unix-like systems just can't have / in filenames
		if strings.Contains(name, "/") {
			return false
		}
	}

	return true
}

// isValidUUID checks if a string is a valid UUID
func isValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}

// getFileExtension returns the file extension without the dot
func getFileExtension(filename string) string {
	ext := filepath.Ext(filename)
	if ext == "" {
		return ""
	}
	return strings.TrimPrefix(ext, ".")
}

// countItems counts the number of items in a directory
func countItems(dirPath string) int {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return 0
	}
	return len(entries)
}

// formatSize formats a file size in bytes to a human-readable string
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatLastModified formats a time.Time to a human-readable string
func formatLastModified(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return "Just now"
	} else if diff < time.Hour {
		minutes := int(diff.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if diff < 48*time.Hour {
		return "Yesterday"
	} else if diff < 7*24*time.Hour {
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%d days ago", days)
	} else if diff < 30*24*time.Hour {
		weeks := int(diff.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	} else {
		return t.Format("Jan 2, 2006")
	}
}

// isRefrigerated checks if a file is refrigerated (compressed)
func isRefrigerated(path string) bool {
	// In a real implementation, you would check if the file is compressed
	// For now, just check if it has a .ref extension
	return strings.HasSuffix(path, ".ref")
}

// CopyItem copies a file or folder to another location

// CopyItem copies a file or folder to another location
func (fb *FileBrowser) CopyItem(srcPath string, destPath string) error {
	sourcePath := fb.GetCurrentPath(srcPath)
	destFullPath := fb.GetCurrentPath(destPath)

	// Check if source exists
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("source does not exist: %w", err)
	}

	// Get the source base name
	baseName := filepath.Base(sourcePath)
	targetPath := filepath.Join(destFullPath, baseName)

	// Check if the destination already exists
	if _, err := os.Stat(targetPath); err == nil {
		return fmt.Errorf("destination already exists: %s", destPath)
	}

	// Check if this is a distributed file
	isDistributed, fileID, err := fb.isDistributedFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to check if file is distributed: %w", err)
	}

	// Copy file or directory
	if sourceInfo.IsDir() {
		return copyDir(sourcePath, targetPath)
	}

	// Copy regular file
	if err := copyFile(sourcePath, targetPath); err != nil {
		return err
	}

	// If it's a distributed file, copy the database entry
	if isDistributed {
		// Generate new file ID for the copy
		newFileID := uuid.New().String()

		db, err := sql.Open("sqlite3", fb.DbPath)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer db.Close()

		// Get original file info
		var fileName string
		var fileSize int64
		var createdAt, updatedAt time.Time
		var totalChunks int

		err = db.QueryRow(
			"SELECT file_name, file_size, created_at, updated_at, total_chunks FROM files WHERE file_id = ?",
			fileID,
		).Scan(&fileName, &fileSize, &createdAt, &updatedAt, &totalChunks)
		if err != nil {
			return fmt.Errorf("failed to get file info: %w", err)
		}

		// Calculate new relative path
		newRelativePath, err := filepath.Rel(fb.KubeLoadsDir, targetPath)
		if err != nil {
			return fmt.Errorf("failed to calculate relative path: %w", err)
		}

		// Insert new file record
		now := time.Now()
		_, err = db.Exec(
			"INSERT INTO files (file_id, file_name, file_path, file_size, created_at, updated_at, is_distributed, total_chunks) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			newFileID, fileName, newRelativePath, fileSize, now, now, true, totalChunks,
		)
		if err != nil {
			return fmt.Errorf("failed to insert copy file record: %w", err)
		}
	}

	return nil
}

// MoveItem moves a file or folder to another location
func (fb *FileBrowser) MoveItem(srcPath string, destPath string) error {
	sourcePath := fb.GetCurrentPath(srcPath)
	destFullPath := fb.GetCurrentPath(destPath)

	// Check if source exists
	_, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("source does not exist: %w", err)
	}

	// Get the source base name
	baseName := filepath.Base(sourcePath)
	targetPath := filepath.Join(destFullPath, baseName)

	// Check if the destination already exists
	if _, err := os.Stat(targetPath); err == nil {
		return fmt.Errorf("destination already exists: %s", destPath)
	}

	// Check if this is a distributed file
	isDistributed, fileID, err := fb.isDistributedFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to check if file is distributed: %w", err)
	}

	// Move the item (rename works across different directories on the same filesystem)
	if err := os.Rename(sourcePath, targetPath); err != nil {
		return fmt.Errorf("failed to move item: %w", err)
	}

	// If it's a distributed file, update the database entry
	if isDistributed {
		db, err := sql.Open("sqlite3", fb.DbPath)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer db.Close()

		// Calculate new relative path
		newRelativePath, err := filepath.Rel(fb.KubeLoadsDir, targetPath)
		if err != nil {
			return fmt.Errorf("failed to calculate relative path: %w", err)
		}

		// Update the file_path in the database
		_, err = db.Exec(
			"UPDATE files SET file_path = ?, updated_at = ? WHERE file_id = ?",
			newRelativePath, time.Now(), fileID,
		)
		if err != nil {
			return fmt.Errorf("failed to update file path in database: %w", err)
		}
	}

	return nil
}

// GetFileInfo returns detailed information about a file or folder
func (fb *FileBrowser) GetFileInfo(path string) (FileType, error) {
	itemPath := fb.GetCurrentPath(path)

	info, err := os.Stat(itemPath)
	if err != nil {
		return FileType{}, fmt.Errorf("failed to access item: %w", err)
	}

	// Calculate relative path from kubeloads directory
	relativePath, err := filepath.Rel(fb.KubeLoadsDir, itemPath)
	if err != nil {
		return FileType{}, fmt.Errorf("failed to calculate relative path: %w", err)
	}

	// Convert path separators to forward slashes for consistent API
	relativePath = "/" + strings.ReplaceAll(relativePath, string(os.PathSeparator), "/")

	// Extract the original filename (remove fileID prefix if present)
	displayName := filepath.Base(itemPath)
	if strings.Contains(displayName, "-") && !info.IsDir() {
		// Try to extract the original filename
		parts := strings.SplitN(displayName, "-", 2)
		if len(parts) == 2 && isValidUUID(parts[0]) {
			displayName = parts[1]
		}
	}

	// Create a FileType object
	file := FileType{
		ID:               uuid.New().String(),
		Name:             displayName,
		Path:             relativePath,
		LastModifiedDate: info.ModTime(),
		LastModified:     formatLastModified(info.ModTime()),
		Owner:            "You", // Default owner
		IsShared:         false,
	}

	if info.IsDir() {
		file.Type = "folder"
		file.ItemCount = countItems(itemPath)
	} else {
		file.Type = "file"
		file.SizeInBytes = info.Size()
		file.Size = formatSize(info.Size())
		file.Extension = getFileExtension(displayName)
		file.IsRefrigerated = isRefrigerated(itemPath)

		// Check if this file is distributed
		isDistributed, _, err := fb.isDistributedFile(path)
		if err != nil {
			log.Printf("Warning: Failed to check distributed status: %v", err)
		} else {
			file.IsDistributed = isDistributed
		}

		if file.IsRefrigerated {
			file.CompressionRatio = 0.5 // Placeholder - would need actual implementation
		}
	}

	return file, nil
}

// Helper functions for copying files and directories

// copyFile copies a single file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Copy file permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}

// copyDir recursively copies a directory tree from src to dst
func copyDir(src, dst string) error {
	// Create the destination directory
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err = os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	// Read the source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursive call for directories
			if err = copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy files
			if err = copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// RefrigerateFile compresses a file to save space (placeholder implementation)
func (fb *FileBrowser) RefrigerateFile(path string) error {
	filePath := fb.GetCurrentPath(path)

	// Check if file exists
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("file does not exist: %w", err)
	}

	if fileInfo.IsDir() {
		return errors.New("cannot refrigerate a directory")
	}

	// In a real implementation, you would compress the file here
	// For now, just add a .ref extension to simulate compression
	refrigeratedPath := filePath + ".ref"

	// Create an empty refrigerated file (placeholder)
	f, err := os.Create(refrigeratedPath)
	if err != nil {
		return fmt.Errorf("failed to create refrigerated file: %w", err)
	}
	defer f.Close()

	// Mark the original file as refrigerated (in a real implementation)
	// For now, we'll just remove the original file
	return os.Remove(filePath)
}

// UnrefrigerateFile decompresses a refrigerated file (placeholder implementation)
func (fb *FileBrowser) UnrefrigerateFile(path string) error {
	filePath := fb.GetCurrentPath(path)

	// In a real implementation, you would check if the file is compressed
	// and decompress it here

	// For now, just remove the .ref extension if it exists
	if strings.HasSuffix(filePath, ".ref") {
		originalPath := strings.TrimSuffix(filePath, ".ref")

		// Create an empty original file (placeholder)
		f, err := os.Create(originalPath)
		if err != nil {
			return fmt.Errorf("failed to create unrefrigerated file: %w", err)
		}
		defer f.Close()

		// Remove the refrigerated file
		return os.Remove(filePath)
	}

	return errors.New("file is not refrigerated")
}

// Helper functions

//MYINITIAL EXAMPLE
// package kfiles

// type FileBrowser struct {
// 	platform_path string
// }

// func LaunchFileBrowser() *FileBrowser {} //this must check and create the kubeloads directory inside platform_path based on os and create a new instance
// func (fn *FileBrowser) CreateFolder() {}
// func (fn *FileBrowser) RenameFolder() {}
// func (fn *FileBrowser) CreateFile()   {}

// //Like this all file operations must be implemented inside kubeloads directory inside FileBrowser
