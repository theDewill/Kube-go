package security

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Kagami/go-face"
	"gocv.io/x/gocv"
)

const (
	// Path to the directory with the model data
	modelDir = "./models"
	// Threshold for face recognition confidence
	recognitionThreshold = 0.6
	// Sample size to train the model
	sampleSize = 5
)

// User represents a registered user in the system
type User struct {
	ID       string
	Email    string
	Samples  []face.Descriptor
	Created  time.Time
	Modified time.Time
}

// FacialSystem is the main struct for the facial recognition system
// that will be exported to the frontend
type FacialSystem struct {
	recognizer       *face.Recognizer
	camera           *gocv.VideoCapture
	users            map[string]*User
	currentUser      *User
	isInitialized    bool
	usersStoragePath string
	mu               sync.Mutex
}

// NewFacialSystem creates a new facial recognition system
func NewFacialSystem(usersPath string) (*FacialSystem, error) {
	// Check if model directory exists
	if _, err := os.Stat(modelDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("model directory '%s' does not exist", modelDir)
	}

	// Initialize face recognizer
	rec, err := face.NewRecognizer(modelDir)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize recognizer: %v", err)
	}

	// We don't initialize the camera here as it will be initialized on demand
	// to avoid keeping it open all the time

	fs := &FacialSystem{
		recognizer:       rec,
		users:            make(map[string]*User),
		isInitialized:    true,
		usersStoragePath: usersPath,
	}

	// Load existing users if available
	if err := fs.loadUsers(); err != nil {
		fmt.Printf("Warning: Could not load users: %v\n", err)
	}

	return fs, nil
}

// loadUsers loads existing users from storage
func (fs *FacialSystem) loadUsers() error {
	// Implementation would load serialized user data
	// This is a placeholder - you'd implement actual persistence
	return nil
}

// saveUsers saves users to storage
func (fs *FacialSystem) saveUsers() error {
	// Implementation would serialize and save user data
	// This is a placeholder - you'd implement actual persistence
	return nil
}

// ensureCameraInitialized makes sure the camera is initialized
func (fs *FacialSystem) ensureCameraInitialized() error {
	if fs.camera != nil {
		return nil
	}

	cam, err := gocv.OpenVideoCapture(0)
	if err != nil {
		return fmt.Errorf("cannot open camera: %v", err)
	}
	fs.camera = cam
	return nil
}

// releaseCamera closes the camera if it's open
func (fs *FacialSystem) releaseCamera() {
	if fs.camera != nil {
		fs.camera.Close()
		fs.camera = nil
	}
}

// Close releases all resources
func (fs *FacialSystem) Close() {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	fs.releaseCamera()
	if fs.recognizer != nil {
		fs.recognizer.Close()
		fs.recognizer = nil
	}
	fs.isInitialized = false
}

// TrainNewUser trains the model with a new user
func (fs *FacialSystem) TrainNewUser(email string) (string, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if !fs.isInitialized {
		return "", fmt.Errorf("facial system not initialized")
	}

	// Initialize camera if needed
	if err := fs.ensureCameraInitialized(); err != nil {
		return "", err
	}

	// Check if user already exists
	for _, user := range fs.users {
		if user.Email == email {
			return "", fmt.Errorf("user with email %s already exists", email)
		}
	}

	// Generate a unique ID for the user
	userID := fmt.Sprintf("user_%d", time.Now().UnixNano())

	fmt.Println("Training model for new user. Please sit in front of the camera.")
	time.Sleep(3 * time.Second) // Give time for user to prepare

	var samples []face.Descriptor

	// Capture multiple samples for better recognition
	for i := 0; i < sampleSize; i++ {
		fmt.Printf("Capturing sample %d/%d...\n", i+1, sampleSize)

		// Capture frame
		img := gocv.NewMat()
		if ok := fs.camera.Read(&img); !ok {
			img.Close()
			return "", fmt.Errorf("cannot read from camera")
		}

		// Save frame temporarily
		tempFile := fmt.Sprintf("temp_sample_%d.jpg", i)
		if ok := gocv.IMWrite(tempFile, img); !ok {
			img.Close()
			return "", fmt.Errorf("failed to save image")
		}
		img.Close()

		// Recognize faces in the image
		faces, err := fs.recognizer.RecognizeFile(tempFile)
		os.Remove(tempFile) // Clean up temp file

		if err != nil {
			return "", fmt.Errorf("recognition error: %v", err)
		}

		if len(faces) == 0 {
			fmt.Println("No face detected. Please make sure your face is visible.")
			i-- // Retry this sample
			time.Sleep(1 * time.Second)
			continue
		}

		if len(faces) > 1 {
			fmt.Println("Multiple faces detected. Please ensure only one person is in view.")
			i-- // Retry this sample
			time.Sleep(1 * time.Second)
			continue
		}

		// Add face descriptor to samples
		samples = append(samples, faces[0].Descriptor)
		time.Sleep(500 * time.Millisecond) // Small delay between captures
	}

	if len(samples) == 0 {
		return "", fmt.Errorf("couldn't capture any valid face samples")
	}

	// Create and store the new user
	newUser := &User{
		ID:       userID,
		Email:    email,
		Samples:  samples,
		Created:  time.Now(),
		Modified: time.Now(),
	}

	fs.users[userID] = newUser

	// Save users to storage
	if err := fs.saveUsers(); err != nil {
		fmt.Printf("Warning: Could not save users: %v\n", err)
	}

	// Release camera after training
	fs.releaseCamera()

	fmt.Println("User successfully registered!")
	return userID, nil
}

// LoginUser tries to authenticate a user using facial recognition
func (fs *FacialSystem) LoginUser() (string, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if !fs.isInitialized {
		return "", fmt.Errorf("facial system not initialized")
	}

	if len(fs.users) == 0 {
		return "", fmt.Errorf("no users registered in the system")
	}

	// Initialize camera if needed
	if err := fs.ensureCameraInitialized(); err != nil {
		return "", err
	}

	// Capture frame
	img := gocv.NewMat()
	defer img.Close()

	if ok := fs.camera.Read(&img); !ok {
		return "", fmt.Errorf("cannot read from camera")
	}

	// Save frame temporarily
	tempFile := "temp_login.jpg"
	if ok := gocv.IMWrite(tempFile, img); !ok {
		return "", fmt.Errorf("failed to save image")
	}

	// Recognize faces in the image
	faces, err := fs.recognizer.RecognizeFile(tempFile)
	os.Remove(tempFile) // Clean up temp file

	if err != nil {
		return "", fmt.Errorf("recognition error: %v", err)
	}

	if len(faces) == 0 {
		return "", fmt.Errorf("no face detected")
	}

	if len(faces) > 1 {
		return "", fmt.Errorf("multiple faces detected")
	}

	// Check the detected face against all registered users
	detectedFace := faces[0].Descriptor
	bestMatch := ""
	bestDistance := float32(100.0) // Initialize with a large value

	for userID, user := range fs.users {
		for _, sample := range user.Samples {
			dist := euclideanDistance(sample, detectedFace)
			if dist < bestDistance {
				bestDistance = dist
				bestMatch = userID
			}
		}
	}

	// Release camera after login attempt
	fs.releaseCamera()

	if bestDistance < recognitionThreshold {
		fs.currentUser = fs.users[bestMatch]
		return fs.users[bestMatch].Email, nil
	}

	return "", fmt.Errorf("face not recognized")
}

// Calculate Euclidean distance between two face descriptors
func euclideanDistance(a, b face.Descriptor) float32 {
	var sum float32
	for i := 0; i < len(a); i++ {
		diff := a[i] - b[i]
		sum += diff * diff
	}
	return sum
}

// GetCurrentUser returns the current logged-in user
func (fs *FacialSystem) GetCurrentUser() (string, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if fs.currentUser == nil {
		return "", fmt.Errorf("no user is currently logged in")
	}

	return fs.currentUser.Email, nil
}

// Logout logs out the current user
func (fs *FacialSystem) Logout() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	fs.currentUser = nil
	return nil
}

// IsInitialized returns whether the system is initialized
func (fs *FacialSystem) IsInitialized() bool {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	return fs.isInitialized
}

// GetRegisteredUserCount returns the number of registered users
func (fs *FacialSystem) GetRegisteredUserCount() int {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	return len(fs.users)
}
