package security

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"sync"
	"time"

	"kube-go/dbops"

	"gocv.io/x/gocv"
)

const (
	// Sample size to train the model
	sampleSize = 5
	// Minimum confidence score for face detection
	minConfidence = 0.7
	// Minimum similarity score for face recognition
	minSimilarity = 0.7
)

// FacialSystem is the main struct for the facial recognition system
type FacialSystem struct {
	faceDetector  gocv.Net
	camera        *gocv.VideoCapture
	currentUser   string
	isInitialized bool
	modelsPath    string
	mu            sync.Mutex
}

// NewFacialSystem creates a new facial recognition system
func NewFacialSystem(dbPath string, modelsPath string) (*FacialSystem, error) {
	// Check if models directory exists
	if _, err := os.Stat(modelsPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("models directory '%s' does not exist", modelsPath)
	}

	// Initialize face detector using OpenCV DNN
	faceProto := filepath.Join(modelsPath, "deploy.prototxt")
	faceModel := filepath.Join(modelsPath, "res10_300x300_ssd_iter_140000.caffemodel")

	if _, err := os.Stat(faceProto); os.IsNotExist(err) {
		return nil, fmt.Errorf("face detection model prototxt not found at %s", faceProto)
	}

	if _, err := os.Stat(faceModel); os.IsNotExist(err) {
		return nil, fmt.Errorf("face detection model not found at %s", faceModel)
	}

	// Initialize database
	dbConfig := dbops.DBConfig{
		Path: dbPath,
	}
	if err := dbops.Initialize(dbConfig); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %v", err)
	}

	// Load the face detection model
	faceNet := gocv.ReadNet(faceModel, faceProto)
	if faceNet.Empty() {
		return nil, fmt.Errorf("error reading face detection model")
	}

	fs := &FacialSystem{
		faceDetector:  faceNet,
		isInitialized: true,
		modelsPath:    modelsPath,
	}

	return fs, nil
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

// detectFace detects a face in the given image and returns the face region
func (fs *FacialSystem) detectFace(img gocv.Mat) (bool, gocv.Mat, error) {
	// Convert to blob for neural network
	blob := gocv.BlobFromImage(img, 1.0, image.Pt(300, 300), gocv.NewScalar(104, 177, 123, 0), false, false)
	defer blob.Close()

	// Set input and run network
	fs.faceDetector.SetInput(blob, "data")
	detections := fs.faceDetector.Forward("detection_out")
	defer detections.Close()

	// Process results
	rows := detections.Rows()
	imgWidth := img.Cols()
	imgHeight := img.Rows()

	var maxConfidence float32
	var bestDetection gocv.Mat
	hasFace := false

	for i := 0; i < rows; i++ {
		confidence := detections.GetFloatAt(i, 2)

		if confidence > minConfidence {
			x1 := int(detections.GetFloatAt(i, 3) * float32(imgWidth))
			y1 := int(detections.GetFloatAt(i, 4) * float32(imgHeight))
			x2 := int(detections.GetFloatAt(i, 5) * float32(imgWidth))
			y2 := int(detections.GetFloatAt(i, 6) * float32(imgHeight))

			// Ensure coordinates are within image boundaries
			if x1 < 0 {
				x1 = 0
			}
			if y1 < 0 {
				y1 = 0
			}
			if x2 >= imgWidth {
				x2 = imgWidth - 1
			}
			if y2 >= imgHeight {
				y2 = imgHeight - 1
			}

			// Only process if we have a valid region
			if x2 > x1 && y2 > y1 {
				// If this is the most confident detection so far
				if confidence > maxConfidence {
					// Close previous best detection if it exists
					if !bestDetection.Empty() {
						bestDetection.Close()
					}

					// Extract face region
					rect := image.Rect(x1, y1, x2, y2)
					face := img.Region(rect)
					bestDetection = face
					maxConfidence = confidence
					hasFace = true
				}
			}
		}
	}

	if !hasFace {
		return false, gocv.NewMat(), nil
	}

	return true, bestDetection, nil
}

// extractFeatures extracts features from a face image
func (fs *FacialSystem) extractFeatures(faceImg gocv.Mat) ([]float64, error) {
	// Convert to grayscale
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(faceImg, &gray, gocv.ColorBGRToGray)

	// Resize to standard size for consistent comparison
	resized := gocv.NewMat()
	defer resized.Close()
	gocv.Resize(gray, &resized, image.Point{X: 128, Y: 128}, 0, 0, gocv.InterpolationDefault)

	// Apply histogram equalization for lighting invariance
	equalized := gocv.NewMat()
	defer equalized.Close()
	gocv.EqualizeHist(resized, &equalized)

	// Create a feature vector by subsampling pixel values
	// This is a simplified approach - a proper face recognition system would use more sophisticated features
	const sampleStep = 8
	features := make([]float64, 0, (128/sampleStep)*(128/sampleStep))

	for y := 0; y < equalized.Rows(); y += sampleStep {
		for x := 0; x < equalized.Cols(); x += sampleStep {
			val := equalized.GetUCharAt(y, x)
			features = append(features, float64(val)/255.0) // Normalize to [0,1]
		}
	}

	return features, nil
}

// Close releases all resources
func (fs *FacialSystem) Close() {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	fs.releaseCamera()
	fs.faceDetector.Close()
	fs.isInitialized = false

	// Close database connection
	dbops.Close()
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
	existingUser, err := dbops.GetUserByEmail(email)
	if err != nil {
		return "", err
	}

	var userID string

	if existingUser != nil {
		userID = existingUser["id"].(string)
	} else {
		// Create new user
		userID, err = dbops.CreateUser(email)
		if err != nil {
			return "", fmt.Errorf("failed to create user: %v", err)
		}
	}

	fmt.Println("Training model for new user. Please sit in front of the camera.")
	time.Sleep(3 * time.Second) // Give time for user to prepare

	var allFeatures [][]float64
	var tempFiles []string

	// Capture multiple samples for better recognition
	for i := 0; i < sampleSize; i++ {
		fmt.Printf("Capturing sample %d/%d...\n", i+1, sampleSize)

		// Capture frame
		img := gocv.NewMat()
		if ok := fs.camera.Read(&img); !ok {
			img.Close()
			return "", fmt.Errorf("cannot read from camera")
		}

		// Save temp file for debugging (optional)
		tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("face_sample_%d.jpg", i))
		gocv.IMWrite(tempFile, img)
		tempFiles = append(tempFiles, tempFile)

		// Detect face
		hasFace, faceImg, err := fs.detectFace(img)
		img.Close()

		if err != nil {
			return "", fmt.Errorf("face detection error: %v", err)
		}

		if !hasFace {
			fmt.Println("No face detected. Please make sure your face is visible.")
			i-- // Retry this sample
			time.Sleep(1 * time.Second)
			continue
		}

		// Extract features from the face
		features, err := fs.extractFeatures(faceImg)
		faceImg.Close()

		if err != nil {
			return "", fmt.Errorf("feature extraction error: %v", err)
		}

		allFeatures = append(allFeatures, features)
		time.Sleep(500 * time.Millisecond) // Small delay between captures
	}

	// Clean up temporary files
	for _, file := range tempFiles {
		os.Remove(file)
	}

	if len(allFeatures) == 0 {
		return "", fmt.Errorf("couldn't capture any valid face samples")
	}

	// Average features for more robust recognition
	avgFeatures := make([]float64, len(allFeatures[0]))
	for _, features := range allFeatures {
		for i, val := range features {
			avgFeatures[i] += val
		}
	}

	for i := range avgFeatures {
		avgFeatures[i] /= float64(len(allFeatures))
	}

	// Store features in database
	_, err = dbops.StoreFaceFeatures(userID, avgFeatures)
	if err != nil {
		return "", fmt.Errorf("failed to store face features: %v", err)
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

	// Detect face
	hasFace, faceImg, err := fs.detectFace(img)
	if err != nil {
		return "", fmt.Errorf("face detection error: %v", err)
	}

	if !hasFace {
		return "", fmt.Errorf("no face detected")
	}

	// Extract features from the detected face
	features, err := fs.extractFeatures(faceImg)
	faceImg.Close()

	if err != nil {
		return "", fmt.Errorf("feature extraction error: %v", err)
	}

	// Find matching user
	email, err := dbops.GetUserByFaceFeatures(features, minSimilarity)
	if err != nil {
		return "", err
	}

	// Store current user
	fs.currentUser = email

	// Release camera after login attempt
	fs.releaseCamera()

	return email, nil
}

// GetCurrentUser returns the current logged-in user
func (fs *FacialSystem) GetCurrentUser() (string, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if fs.currentUser == "" {
		return "", fmt.Errorf("no user is currently logged in")
	}

	return fs.currentUser, nil
}

// Logout logs out the current user
func (fs *FacialSystem) Logout() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	fs.currentUser = ""
	return nil
}

// IsInitialized returns whether the system is initialized
func (fs *FacialSystem) IsInitialized() bool {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	return fs.isInitialized
}

// GetRegisteredUserCount returns the number of registered users
func (fs *FacialSystem) GetRegisteredUserCount() (int, error) {
	// Get all users from database
	users, err := dbops.GetAllData("users", nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get users: %v", err)
	}

	return len(users), nil
}
