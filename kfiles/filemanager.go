package kfiles

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
)

// FileBrowser manages file operations within the Kube application
type FileBrowser struct {
	PlatformPath string // Base platform-specific path
	KubeLoadsDir string // Dedicated kubeloads directory
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
}

// LaunchFileBrowser creates a new FileBrowser instance with the platform-specific path
func LaunchFileBrowser() (*FileBrowser, error) {
	platformPath, err := getPlatformSpecificPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get platform-specific path: %w", err)
	}

	// Create the kubeloads directory
	kubeLoadsDir := filepath.Join(platformPath, "kubeloads")
	if err := os.MkdirAll(kubeLoadsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create kubeloads directory: %w", err)
	}

	fb := &FileBrowser{
		PlatformPath: platformPath,
		KubeLoadsDir: kubeLoadsDir,
	}
	print("FileBrowser Created")

	return fb, nil
}

// getPlatformSpecificPath returns the appropriate directory path based on the OS
func getPlatformSpecificPath() (string, error) {
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

		// Create a FileType object
		file := FileType{
			ID:               uuid.New().String(),
			Name:             entry.Name(),
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
			file.Extension = getFileExtension(entry.Name())
			file.IsRefrigerated = isRefrigerated(entryPath)
			if file.IsRefrigerated {
				file.CompressionRatio = 0.5 // Placeholder - would need actual implementation
			}
		}

		files = append(files, file)
	}

	return files, nil
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

	return os.Rename(itemPath, newPath)
}

// DeleteItem deletes a file or folder
func (fb *FileBrowser) DeleteItem(path string) error {
	itemPath := fb.GetCurrentPath(path)

	info, err := os.Stat(itemPath)
	if err != nil {
		return fmt.Errorf("failed to access item: %w", err)
	}

	if info.IsDir() {
		return os.RemoveAll(itemPath)
	}

	return os.Remove(itemPath)
}

// UploadFile saves an uploaded file to the specified directory
func (fb *FileBrowser) UploadFile(directoryPath string, fileName string, fileData []byte) error {
	if !isValidName(fileName) {
		return errors.New("invalid file name")
	}

	targetPath := filepath.Join(fb.GetCurrentPath(directoryPath), fileName)

	// Check if file already exists
	if _, err := os.Stat(targetPath); err == nil {
		return fmt.Errorf("file already exists: %s", fileName)
	}

	return os.WriteFile(targetPath, fileData, 0644)
}

// DownloadFile reads a file and returns its contents as bytes
func (fb *FileBrowser) DownloadFile(path string) ([]byte, error) {
	filePath := fb.GetCurrentPath(path)

	_, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to access file: %w", err)
	}

	return os.ReadFile(filePath)
}

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

	// Copy file or directory
	if sourceInfo.IsDir() {
		return copyDir(sourcePath, targetPath)
	}

	return copyFile(sourcePath, targetPath)
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

	// Move the item (rename works across different directories)
	return os.Rename(sourcePath, targetPath)
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

	// Create a FileType object
	file := FileType{
		ID:               uuid.New().String(),
		Name:             filepath.Base(itemPath),
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
		file.Extension = getFileExtension(info.Name())
		file.IsRefrigerated = isRefrigerated(itemPath)
		if file.IsRefrigerated {
			file.CompressionRatio = 0.5 // Placeholder - would need actual implementation
		}
	}

	return file, nil
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
