package security

import (
	"encoding/base64"
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"
	"sync"

	"kube-go/dbops"

	"gocv.io/x/gocv"
)

const (
	// Lower thresholds for better detection
	minSimilarity = 0.790
)

type FacialSystem struct {
	faceClassifier gocv.CascadeClassifier
	currentUser    string
	isInitialized  bool
	mu             sync.Mutex
}

func NewFacialSystem(dbPath string, modelsPath string) (*FacialSystem, error) {
	// Initialize database
	// dbConfig := dbops.DBConfig{Path: dbPath}
	// if err := dbops.Initialize(dbConfig); err != nil {
	// 	return nil, fmt.Errorf("failed to initialize database: %v", err)
	// }

	// Load Haar cascade for face detection
	classifier := gocv.NewCascadeClassifier()

	// Try different cascade files
	cascadeFiles := []string{
		"haarcascade_frontalface_alt.xml",
		"haarcascade_frontalface_default.xml",
		"haarcascade_frontalface_alt2.xml",
	}

	var cascadeLoaded bool
	for _, file := range cascadeFiles {
		cascadePath := filepath.Join(modelsPath, file)
		if classifier.Load(cascadePath) {
			fmt.Printf("Loaded cascade: %s\n", file)
			cascadeLoaded = true
			break
		}
	}

	if !cascadeLoaded {
		// Try loading from OpenCV installation directory
		if classifier.Load("haarcascade_frontalface_alt.xml") {
			cascadeLoaded = true
			fmt.Println("Loaded default cascade from OpenCV")
		}
	}

	if !cascadeLoaded {
		return nil, fmt.Errorf("could not load any face cascade classifier")
	}

	return &FacialSystem{
		faceClassifier: classifier,
		isInitialized:  true,
	}, nil
}

// Simple and reliable face detection
func (fs *FacialSystem) detectFace(img gocv.Mat) (bool, gocv.Mat, error) {
	if img.Empty() {
		return false, gocv.NewMat(), fmt.Errorf("empty image")
	}

	// Convert to grayscale for face detection
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(img, &gray, gocv.ColorBGRToGray)

	// Detect faces with relaxed parameters
	faces := fs.faceClassifier.DetectMultiScaleWithParams(
		gray,
		1.1,                // Scale factor
		3,                  // Min neighbors (lower = more detections)
		0,                  // Flags
		image.Pt(30, 30),   // Min size
		image.Pt(300, 300), // Max size
	)

	if len(faces) == 0 {
		return false, gocv.NewMat(), nil
	}

	// Find the largest face
	largest := faces[0]
	for _, face := range faces {
		if face.Dx()*face.Dy() > largest.Dx()*largest.Dy() {
			largest = face
		}
	}

	// Extract face region from original color image
	faceRegion := img.Region(largest)
	fmt.Printf("Detected face at: %v, size: %dx%d\n", largest, largest.Dx(), largest.Dy())

	return true, faceRegion, nil
}

// Simplified feature extraction
func (fs *FacialSystem) extractFeatures(faceImg gocv.Mat) ([]float64, error) {
	// Convert to grayscale
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(faceImg, &gray, gocv.ColorBGRToGray)

	// Resize to standard size
	resized := gocv.NewMat()
	defer resized.Close()
	gocv.Resize(gray, &resized, image.Point{X: 100, Y: 100}, 0, 0, gocv.InterpolationDefault)

	// Apply histogram equalization
	equalized := gocv.NewMat()
	defer equalized.Close()
	gocv.EqualizeHist(resized, &equalized)

	// Extract pixel values as features (simplified approach)
	features := make([]float64, 0, 100*100)
	for y := 0; y < equalized.Rows(); y++ {
		for x := 0; x < equalized.Cols(); x++ {
			val := equalized.GetUCharAt(y, x)
			features = append(features, float64(val)/255.0)
		}
	}

	return features, nil
}

// Enhanced feature extraction with Local Binary Patterns
func (fs *FacialSystem) extractEnhancedFeatures(faceImg gocv.Mat) ([]float64, error) {
	// Convert to grayscale
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(faceImg, &gray, gocv.ColorBGRToGray)

	// Resize to standard size
	resized := gocv.NewMat()
	defer resized.Close()
	gocv.Resize(gray, &resized, image.Point{X: 100, Y: 100}, 0, 0, gocv.InterpolationDefault)

	// Apply histogram equalization for lighting invariance
	equalized := gocv.NewMat()
	defer equalized.Close()
	gocv.EqualizeHist(resized, &equalized)

	// Apply LBP-like feature extraction (simplified for Go implementation)
	// This creates more discriminative features than raw pixel values
	features := make([]float64, 0, 100*100)

	// For each pixel (excluding borders), compare with 8 neighbors
	for y := 1; y < equalized.Rows()-1; y++ {
		for x := 1; x < equalized.Cols()-1; x++ {
			center := float64(equalized.GetUCharAt(y, x))

			// Extract a simple gradient-based feature
			gradientSum := 0.0
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if dx == 0 && dy == 0 {
						continue
					}
					neighbor := float64(equalized.GetUCharAt(y+dy, x+dx))
					gradientSum += math.Abs(center - neighbor)
				}
			}

			// Normalize and add to features
			features = append(features, gradientSum/8.0/255.0)
		}
	}

	return features, nil
}

func (fs *FacialSystem) TrainNewUserWithFrames(dbpath string, faceFrames []string) (string, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if !fs.isInitialized {
		return "", fmt.Errorf("facial system not initialized")
	}

	if len(faceFrames) == 0 {
		return "", fmt.Errorf("no face frames provided")
	}

	var allFeatures [][]float64
	processedFrames := 0

	// Process each frame
	for i, frameData := range faceFrames {
		fmt.Printf("Processing frame %d/%d\n", i+1, len(faceFrames))

		// Decode base64 image
		imgData, err := base64.StdEncoding.DecodeString(frameData)
		if err != nil {
			fmt.Printf("Failed to decode frame %d: %v\n", i, err)
			continue
		}

		// Convert to OpenCV Mat
		img, err := gocv.IMDecode(imgData, gocv.IMReadColor)
		if err != nil {
			fmt.Printf("Failed to decode image %d: %v\n", i, err)
			continue
		}

		// Save debug image
		debugPath := filepath.Join(os.TempDir(), fmt.Sprintf("debug_frame_%d.jpg", i))
		gocv.IMWrite(debugPath, img)
		fmt.Printf("Debug image saved: %s\n", debugPath)

		// Detect face
		hasFace, faceImg, err := fs.detectFace(img)
		img.Close()

		if err != nil {
			fmt.Printf("Face detection error on frame %d: %v\n", i, err)
			continue
		}

		if !hasFace {
			fmt.Printf("No face detected in frame %d\n", i)
			continue
		}

		// Save detected face for debugging
		facePath := filepath.Join(os.TempDir(), fmt.Sprintf("detected_face_%d.jpg", i))
		gocv.IMWrite(facePath, faceImg)
		fmt.Printf("Detected face saved: %s\n", facePath)

		// Extract features
		features, err := fs.extractFeatures(faceImg)
		faceImg.Close()

		if err != nil {
			fmt.Printf("Feature extraction error on frame %d: %v\n", i, err)
			continue
		}

		allFeatures = append(allFeatures, features)
		processedFrames++
		fmt.Printf("Successfully processed frame %d\n", i)
	}

	if len(allFeatures) == 0 {
		return "", fmt.Errorf("couldn't extract features from any frame (processed %d frames)", processedFrames)
	}

	// Average features for robust recognition
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
	fid, err := dbops.StoreFaceFeatures(dbpath, avgFeatures)
	if err != nil {
		return "", fmt.Errorf("failed to store face features: %v", err)
	}

	fmt.Printf("Successfully registered facial features from %d frames\n", len(allFeatures))
	return fid, nil
}

// Add this method for face-based login
func (fs *FacialSystem) LoginUserWithFrame(dbpath string, frameData string) (string, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if !fs.isInitialized {
		return "", fmt.Errorf("facial system not initialized")
	}

	// Decode image
	imgData, err := base64.StdEncoding.DecodeString(frameData)
	if err != nil {
		return "", fmt.Errorf("failed to decode frame: %v", err)
	}

	img, err := gocv.IMDecode(imgData, gocv.IMReadColor)
	if err != nil {
		return "", fmt.Errorf("failed to decode image: %v", err)
	}
	defer img.Close()

	// Detect face
	hasFace, faceImg, err := fs.detectFace(img)
	if err != nil {
		return "", fmt.Errorf("face detection error: %v", err)
	}

	if !hasFace {
		return "", fmt.Errorf("no face detected")
	}
	defer faceImg.Close()

	// Extract features
	features, err := fs.extractFeatures(faceImg)
	if err != nil {
		return "", fmt.Errorf("feature extraction error: %v", err)
	}

	// Find matching user in database
	email, err := dbops.GetUserByFaceFeatures(dbpath, features, minSimilarity)
	if err != nil {
		return "", err
	}

	fs.currentUser = email
	return email, nil
}

func (fs *FacialSystem) Close() {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	fs.faceClassifier.Close()
	fs.isInitialized = false
	dbops.Close()
}

// Keep other methods the same...
func (fs *FacialSystem) GetCurrentUser() (string, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if fs.currentUser == "" {
		return "", fmt.Errorf("no user is currently logged in")
	}
	return fs.currentUser, nil
}

func (fs *FacialSystem) Logout() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.currentUser = ""
	return nil
}

func (fs *FacialSystem) IsInitialized() bool {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return fs.isInitialized
}

//<-----------------------------------OLD
// package security

// import (
// 	"encoding/base64"
// 	"fmt"
// 	"image"
// 	"os"
// 	"path/filepath"
// 	"sync"
// 	"time"

// 	"kube-go/dbops"

// 	"gocv.io/x/gocv"
// )

// const (
// 	// Sample size to train the model
// 	sampleSize = 5
// 	// Minimum confidence score for face detection
// 	minConfidence = 0.4
// 	// Minimum similarity score for face recognition
// 	minSimilarity = 0.7
// )

// // FacialSystem is the main struct for the facial recognition system
// type FacialSystem struct {
// 	faceDetector  gocv.Net
// 	camera        *gocv.VideoCapture
// 	currentUser   string
// 	isInitialized bool
// 	modelsPath    string
// 	mu            sync.Mutex
// }

// // NewFacialSystem creates a new facial recognition system
// func NewFacialSystem(dbPath string, modelsPath string) (*FacialSystem, error) {
// 	// Check if models directory exists
// 	if _, err := os.Stat(modelsPath); os.IsNotExist(err) {
// 		return nil, fmt.Errorf("models directory '%s' does not exist", modelsPath)
// 	}

// 	// Initialize face detector using OpenCV DNN
// 	faceProto := filepath.Join(modelsPath, "deploy.prototxt")
// 	faceModel := filepath.Join(modelsPath, "res10_300x300_ssd_iter_140000.caffemodel")

// 	if _, err := os.Stat(faceProto); os.IsNotExist(err) {
// 		return nil, fmt.Errorf("face detection model prototxt not found at %s", faceProto)
// 	}

// 	if _, err := os.Stat(faceModel); os.IsNotExist(err) {
// 		return nil, fmt.Errorf("face detection model not found at %s", faceModel)
// 	}

// 	// Initialize database
// 	dbConfig := dbops.DBConfig{
// 		Path: dbPath,
// 	}
// 	if err := dbops.Initialize(dbConfig); err != nil {
// 		return nil, fmt.Errorf("failed to initialize database: %v", err)
// 	}

// 	// Load the face detection model
// 	faceNet := gocv.ReadNet(faceModel, faceProto)
// 	if faceNet.Empty() {
// 		return nil, fmt.Errorf("error reading face detection model")
// 	}

// 	fs := &FacialSystem{
// 		faceDetector:  faceNet,
// 		isInitialized: true,
// 		modelsPath:    modelsPath,
// 	}

// 	return fs, nil
// }

// // ensureCameraInitialized makes sure the camera is initialized
// func (fs *FacialSystem) ensureCameraInitialized() error {
// 	if fs.camera != nil {
// 		return nil
// 	}

// 	cam, err := gocv.OpenVideoCapture(0)
// 	if err != nil {
// 		return fmt.Errorf("cannot open camera: %v", err)
// 	}
// 	fs.camera = cam
// 	return nil
// }

// // releaseCamera closes the camera if it's open
// func (fs *FacialSystem) releaseCamera() {
// 	if fs.camera != nil {
// 		fs.camera.Close()
// 		fs.camera = nil
// 	}
// }

// // detectFace detects a face in the given image and returns the face region
// func (fs *FacialSystem) detectFaceOld(img gocv.Mat) (bool, gocv.Mat, error) {
// 	// Convert to blob for neural network
// 	blob := gocv.BlobFromImage(img, 1.0, image.Pt(300, 300), gocv.NewScalar(104, 177, 123, 0), false, false)
// 	defer blob.Close()

// 	// Set input and run network
// 	fs.faceDetector.SetInput(blob, "data")
// 	detections := fs.faceDetector.Forward("detection_out")
// 	defer detections.Close()

// 	// Process results
// 	rows := detections.Rows()
// 	imgWidth := img.Cols()
// 	imgHeight := img.Rows()

// 	var maxConfidence float32
// 	var bestDetection gocv.Mat
// 	hasFace := false

// 	for i := 0; i < rows; i++ {
// 		confidence := detections.GetFloatAt(i, 2)

// 		if confidence > minConfidence {
// 			x1 := int(detections.GetFloatAt(i, 3) * float32(imgWidth))
// 			y1 := int(detections.GetFloatAt(i, 4) * float32(imgHeight))
// 			x2 := int(detections.GetFloatAt(i, 5) * float32(imgWidth))
// 			y2 := int(detections.GetFloatAt(i, 6) * float32(imgHeight))

// 			// Ensure coordinates are within image boundaries
// 			if x1 < 0 {
// 				x1 = 0
// 			}
// 			if y1 < 0 {
// 				y1 = 0
// 			}
// 			if x2 >= imgWidth {
// 				x2 = imgWidth - 1
// 			}
// 			if y2 >= imgHeight {
// 				y2 = imgHeight - 1
// 			}

// 			// Only process if we have a valid region
// 			if x2 > x1 && y2 > y1 {
// 				// If this is the most confident detection so far
// 				if confidence > maxConfidence {
// 					// Close previous best detection if it exists
// 					if !bestDetection.Empty() {
// 						bestDetection.Close()
// 					}

// 					// Extract face region
// 					rect := image.Rect(x1, y1, x2, y2)
// 					face := img.Region(rect)
// 					bestDetection = face
// 					maxConfidence = confidence
// 					hasFace = true
// 				}
// 			}
// 		}
// 	}

// 	if !hasFace {
// 		return false, gocv.NewMat(), nil
// 	}

// 	return true, bestDetection, nil
// }

// // detectFace [DEBIG]
// func (fs *FacialSystem) detectFaceDebug(img gocv.Mat) (bool, gocv.Mat, error) {
// 	// Log image dimensions for debugging
// 	fmt.Printf("Input image dimensions: %dx%d\n", img.Cols(), img.Rows())

// 	// Check if image is empty
// 	if img.Empty() {
// 		return false, gocv.NewMat(), fmt.Errorf("input image is empty")
// 	}

// 	// Convert to blob for neural network
// 	blob := gocv.BlobFromImage(img, 1.0, image.Pt(300, 300), gocv.NewScalar(104, 177, 123, 0), false, false)
// 	defer blob.Close()

// 	// Set input and run network
// 	fs.faceDetector.SetInput(blob, "data")
// 	detections := fs.faceDetector.Forward("detection_out")
// 	defer detections.Close()

// 	// Log detection results
// 	fmt.Printf("Detection tensor dimensions: %dx%dx%dx%d\n",
// 		detections.Size()[0], detections.Size()[1], detections.Size()[2], detections.Size()[3])

// 	// Process results
// 	rows := detections.Rows()
// 	imgWidth := img.Cols()
// 	imgHeight := img.Rows()

// 	fmt.Printf("Number of detection rows: %d\n", rows)

// 	var maxConfidence float32
// 	var bestDetection gocv.Mat
// 	hasFace := false

// 	for i := 0; i < rows; i++ {
// 		confidence := detections.GetFloatAt(i, 2)
// 		fmt.Printf("Detection %d confidence: %.3f (threshold: %.3f)\n", i, confidence, minConfidence)

// 		if confidence > minConfidence {
// 			x1 := int(detections.GetFloatAt(i, 3) * float32(imgWidth))
// 			y1 := int(detections.GetFloatAt(i, 4) * float32(imgHeight))
// 			x2 := int(detections.GetFloatAt(i, 5) * float32(imgWidth))
// 			y2 := int(detections.GetFloatAt(i, 6) * float32(imgHeight))

// 			fmt.Printf("Raw coordinates: (%d,%d) to (%d,%d)\n", x1, y1, x2, y2)

// 			// Ensure coordinates are within image boundaries
// 			if x1 < 0 {
// 				x1 = 0
// 			}
// 			if y1 < 0 {
// 				y1 = 0
// 			}
// 			if x2 >= imgWidth {
// 				x2 = imgWidth - 1
// 			}
// 			if y2 >= imgHeight {
// 				y2 = imgHeight - 1
// 			}

// 			fmt.Printf("Adjusted coordinates: (%d,%d) to (%d,%d)\n", x1, y1, x2, y2)

// 			// Only process if we have a valid region
// 			if x2 > x1 && y2 > y1 {
// 				// If this is the most confident detection so far
// 				if confidence > maxConfidence {
// 					// Close previous best detection if it exists
// 					if !bestDetection.Empty() {
// 						bestDetection.Close()
// 					}

// 					// Extract face region
// 					rect := image.Rect(x1, y1, x2, y2)
// 					face := img.Region(rect)
// 					bestDetection = face
// 					maxConfidence = confidence
// 					hasFace = true
// 					fmt.Printf("Found face with confidence %.3f\n", confidence)
// 				}
// 			} else {
// 				fmt.Printf("Invalid face region: width=%d, height=%d\n", x2-x1, y2-y1)
// 			}
// 		}
// 	}

// 	if !hasFace {
// 		fmt.Println("No face detected above confidence threshold")
// 		return false, gocv.NewMat(), nil
// 	}

// 	fmt.Printf("Best face detection confidence: %.3f\n", maxConfidence)
// 	return true, bestDetection, nil
// }

// // Ultra-safe fallback version
// func (fs *FacialSystem) detectFace(img gocv.Mat) (bool, gocv.Mat, error) {
// 	// Log image dimensions for debugging
// 	fmt.Printf("Input image dimensions: %dx%d\n", img.Cols(), img.Rows())

// 	// Check if image is empty
// 	if img.Empty() {
// 		return false, gocv.NewMat(), fmt.Errorf("input image is empty")
// 	}

// 	// Convert to blob for neural network
// 	blob := gocv.BlobFromImage(img, 1.0, image.Pt(300, 300), gocv.NewScalar(104, 177, 123, 0), false, false)
// 	defer blob.Close()

// 	// Set input and run network
// 	fs.faceDetector.SetInput(blob, "data")
// 	detections := fs.faceDetector.Forward("detection_out")
// 	defer detections.Close()

// 	// Log detection results
// 	sizes := detections.Size()
// 	fmt.Printf("Detection tensor dimensions: %dx%dx%dx%d\n", sizes[0], sizes[1], sizes[2], sizes[3])

// 	// Let's try the original approach but with a safety check
// 	rows := detections.Rows()
// 	fmt.Printf("Detections.Rows(): %d\n", rows)

// 	if rows <= 0 {
// 		// If Rows() doesn't work, try a simple fallback
// 		fmt.Println("Cannot determine number of detections - using fallback")
// 		return false, gocv.NewMat(), nil
// 	}

// 	imgWidth := img.Cols()
// 	imgHeight := img.Rows()

// 	var maxConfidence float32
// 	var bestDetection gocv.Mat
// 	hasFace := false

// 	// Use the original method with safety checks
// 	for i := 0; i < rows; i++ {
// 		// Add bounds checking before accessing
// 		confidence := detections.GetFloatAt(i, 2)
// 		fmt.Printf("Detection %d confidence: %.3f (threshold: %.3f)\n", i, confidence, minConfidence)

// 		if confidence > minConfidence {
// 			x1 := int(detections.GetFloatAt(i, 3) * float32(imgWidth))
// 			y1 := int(detections.GetFloatAt(i, 4) * float32(imgHeight))
// 			x2 := int(detections.GetFloatAt(i, 5) * float32(imgWidth))
// 			y2 := int(detections.GetFloatAt(i, 6) * float32(imgHeight))

// 			// Ensure coordinates are within image boundaries
// 			if x1 < 0 {
// 				x1 = 0
// 			}
// 			if y1 < 0 {
// 				y1 = 0
// 			}
// 			if x2 >= imgWidth {
// 				x2 = imgWidth - 1
// 			}
// 			if y2 >= imgHeight {
// 				y2 = imgHeight - 1
// 			}

// 			// Only process if we have a valid region
// 			if x2 > x1 && y2 > y1 {
// 				width := x2 - x1
// 				height := y2 - y1

// 				// If this is the most confident detection so far
// 				if confidence > maxConfidence && width > 20 && height > 20 {
// 					// Close previous best detection if it exists
// 					if !bestDetection.Empty() {
// 						bestDetection.Close()
// 					}

// 					// Extract face region
// 					rect := image.Rect(x1, y1, x2, y2)
// 					face := img.Region(rect)
// 					bestDetection = face
// 					maxConfidence = confidence
// 					hasFace = true
// 					fmt.Printf("Found face with confidence %.3f\n", confidence)
// 				}
// 			}
// 		}
// 	}

// 	if !hasFace {
// 		fmt.Println("No face detected above confidence threshold")
// 		return false, gocv.NewMat(), nil
// 	}

// 	return true, bestDetection, nil
// }

// // extractFeatures extracts features from a face image
// func (fs *FacialSystem) extractFeatures(faceImg gocv.Mat) ([]float64, error) {
// 	// Convert to grayscale
// 	gray := gocv.NewMat()
// 	defer gray.Close()
// 	gocv.CvtColor(faceImg, &gray, gocv.ColorBGRToGray)

// 	// Resize to standard size for consistent comparison
// 	resized := gocv.NewMat()
// 	defer resized.Close()
// 	gocv.Resize(gray, &resized, image.Point{X: 128, Y: 128}, 0, 0, gocv.InterpolationDefault)

// 	// Apply histogram equalization for lighting invariance
// 	equalized := gocv.NewMat()
// 	defer equalized.Close()
// 	gocv.EqualizeHist(resized, &equalized)

// 	// Create a feature vector by subsampling pixel values
// 	// This is a simplified approach - a proper face recognition system would use more sophisticated features
// 	const sampleStep = 8
// 	features := make([]float64, 0, (128/sampleStep)*(128/sampleStep))

// 	for y := 0; y < equalized.Rows(); y += sampleStep {
// 		for x := 0; x < equalized.Cols(); x += sampleStep {
// 			val := equalized.GetUCharAt(y, x)
// 			features = append(features, float64(val)/255.0) // Normalize to [0,1]
// 		}
// 	}

// 	return features, nil
// }

// // Close releases all resources
// func (fs *FacialSystem) Close() {
// 	fs.mu.Lock()
// 	defer fs.mu.Unlock()

// 	fs.releaseCamera()
// 	fs.faceDetector.Close()
// 	fs.isInitialized = false

// 	// Close database connection
// 	dbops.Close()
// }

// // TrainNewUser trains the model with a new user
// func (fs *FacialSystem) TrainNewUser(email string) (string, error) {
// 	fs.mu.Lock()
// 	defer fs.mu.Unlock()

// 	if !fs.isInitialized {
// 		return "", fmt.Errorf("facial system not initialized")
// 	}

// 	// Initialize camera if needed
// 	if err := fs.ensureCameraInitialized(); err != nil {
// 		return "", err
// 	}

// 	// Check if user already exists
// 	// existingUser, err := dbops.GetUserByEmail(email)
// 	// if err != nil {
// 	// 	return "", err
// 	// }

// 	// var userID string

// 	// if existingUser != nil {
// 	// 	userID = existingUser["id"].(string)
// 	// } else {
// 	// 	// Create new user
// 	// 	userID, err = dbops.CreateUser(email)
// 	// 	if err != nil {
// 	// 		return "", fmt.Errorf("failed to create user: %v", err)
// 	// 	}
// 	// }

// 	fmt.Println("Training model for new user. Please sit in front of the camera.")
// 	time.Sleep(3 * time.Second) // Give time for user to prepare

// 	var allFeatures [][]float64
// 	var tempFiles []string

// 	// Capture multiple samples for better recognition
// 	for i := 0; i < sampleSize; i++ {
// 		fmt.Printf("Capturing sample %d/%d...\n", i+1, sampleSize)

// 		// Capture frame
// 		img := gocv.NewMat()
// 		if ok := fs.camera.Read(&img); !ok {
// 			img.Close()
// 			return "", fmt.Errorf("cannot read from camera")
// 		}

// 		// Save temp file for debugging (optional)
// 		tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("face_sample_%d.jpg", i))
// 		gocv.IMWrite(tempFile, img)
// 		tempFiles = append(tempFiles, tempFile)

// 		// Detect face
// 		hasFace, faceImg, err := fs.detectFace(img)
// 		img.Close()

// 		if err != nil {
// 			return "", fmt.Errorf("face detection error: %v", err)
// 		}

// 		if !hasFace {
// 			fmt.Println("No face detected. Please make sure your face is visible.")
// 			i-- // Retry this sample
// 			time.Sleep(1 * time.Second)
// 			continue
// 		}

// 		// Extract features from the face
// 		features, err := fs.extractFeatures(faceImg)
// 		faceImg.Close()

// 		if err != nil {
// 			return "", fmt.Errorf("feature extraction error: %v", err)
// 		}

// 		allFeatures = append(allFeatures, features)
// 		time.Sleep(500 * time.Millisecond) // Small delay between captures
// 	}

// 	// Clean up temporary files
// 	for _, file := range tempFiles {
// 		os.Remove(file)
// 	}

// 	if len(allFeatures) == 0 {
// 		return "", fmt.Errorf("couldn't capture any valid face samples")
// 	}

// 	// Average features for more robust recognition
// 	avgFeatures := make([]float64, len(allFeatures[0]))
// 	for _, features := range allFeatures {
// 		for i, val := range features {
// 			avgFeatures[i] += val
// 		}
// 	}

// 	for i := range avgFeatures {
// 		avgFeatures[i] /= float64(len(allFeatures))
// 	}

// 	// Store features in database
// 	fid, err := dbops.StoreFaceFeatures(avgFeatures)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to store face features: %v", err)
// 	}

// 	// Release camera after training
// 	fs.releaseCamera()

// 	fmt.Println("User successfully registered!")
// 	return fid, nil
// }

// func (fs *FacialSystem) TrainNewUserWithFrames(faceFrames []string) (string, error) {
// 	fs.mu.Lock()
// 	defer fs.mu.Unlock()

// 	if !fs.isInitialized {
// 		return "", fmt.Errorf("facial system not initialized")
// 	}

// 	if len(faceFrames) == 0 {
// 		return "", fmt.Errorf("no face frames provided")
// 	}

// 	// Check if user already exists
// 	// existingUser, err := dbops.GetUserByEmail(email)
// 	// if err != nil {
// 	// 	return "", err
// 	// }

// 	// // var userID string
// 	// // if existingUser != nil {
// 	// // 	userID = existingUser["id"].(string)
// 	// // } else {
// 	// // 	// Create new user
// 	// // 	userID, err = dbops.CreateUser(email)
// 	// // 	if err != nil {
// 	// // 		return "", fmt.Errorf("failed to create user: %v", err)
// 	// // 	}
// 	// // }

// 	var allFeatures [][]float64

// 	// Process each frame
// 	for i, frameData := range faceFrames {
// 		// Decode base64 image
// 		imgData, err := base64.StdEncoding.DecodeString(frameData)
// 		if err != nil {
// 			return "", fmt.Errorf("failed to decode frame %d: %v", i, err)
// 		}

// 		// Convert to OpenCV Mat
// 		img, err := gocv.IMDecode(imgData, gocv.IMReadColor)
// 		if err != nil {
// 			return "", fmt.Errorf("failed to decode image %d: %v", i, err)
// 		}

// 		// Detect face
// 		hasFace, faceImg, err := fs.detectFace(img)
// 		img.Close()

// 		if err != nil {
// 			return "", fmt.Errorf("face detection error on frame %d: %v", i, err)
// 		}

// 		if !hasFace {
// 			fmt.Printf("No face detected in frame %d\n", i)
// 			continue
// 		}

// 		// Extract features from the face
// 		features, err := fs.extractFeatures(faceImg)
// 		faceImg.Close()

// 		if err != nil {
// 			return "", fmt.Errorf("feature extraction error on frame %d: %v", i, err)
// 		}

// 		allFeatures = append(allFeatures, features)
// 	}

// 	if len(allFeatures) == 0 {
// 		return "", fmt.Errorf("couldn't extract features from any frame")
// 	}

// 	// Average features for more robust recognition
// 	avgFeatures := make([]float64, len(allFeatures[0]))
// 	for _, features := range allFeatures {
// 		for i, val := range features {
// 			avgFeatures[i] += val
// 		}
// 	}

// 	for i := range avgFeatures {
// 		avgFeatures[i] /= float64(len(allFeatures))
// 	}

// 	// Store features in database
// 	fid, err := dbops.StoreFaceFeatures(avgFeatures)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to store face features: %v", err)
// 	}

// 	fmt.Println("features successfully registered!")
// 	return fid, nil
// }

// // LoginUser tries to authenticate a user using facial recognition
// func (fs *FacialSystem) LoginUser() (string, error) {
// 	fs.mu.Lock()
// 	defer fs.mu.Unlock()

// 	if !fs.isInitialized {
// 		return "", fmt.Errorf("facial system not initialized")
// 	}

// 	// Initialize camera if needed
// 	if err := fs.ensureCameraInitialized(); err != nil {
// 		return "", err
// 	}

// 	// Capture frame
// 	img := gocv.NewMat()
// 	defer img.Close()

// 	if ok := fs.camera.Read(&img); !ok {
// 		return "", fmt.Errorf("cannot read from camera")
// 	}

// 	// Detect face
// 	hasFace, faceImg, err := fs.detectFace(img)
// 	if err != nil {
// 		return "", fmt.Errorf("face detection error: %v", err)
// 	}

// 	if !hasFace {
// 		return "", fmt.Errorf("no face detected")
// 	}

// 	// Extract features from the detected face
// 	features, err := fs.extractFeatures(faceImg)
// 	faceImg.Close()

// 	if err != nil {
// 		return "", fmt.Errorf("feature extraction error: %v", err)
// 	}

// 	// Find matching user
// 	email, err := dbops.GetUserByFaceFeatures(features, minSimilarity)
// 	if err != nil {
// 		return "", err
// 	}

// 	// Store current user
// 	fs.currentUser = email

// 	// Release camera after login attempt
// 	fs.releaseCamera()

// 	return email, nil
// }

// // GetCurrentUser returns the current logged-in user
// func (fs *FacialSystem) GetCurrentUser() (string, error) {
// 	fs.mu.Lock()
// 	defer fs.mu.Unlock()

// 	if fs.currentUser == "" {
// 		return "", fmt.Errorf("no user is currently logged in")
// 	}

// 	return fs.currentUser, nil
// }

// // Logout logs out the current user
// func (fs *FacialSystem) Logout() error {
// 	fs.mu.Lock()
// 	defer fs.mu.Unlock()

// 	fs.currentUser = ""
// 	return nil
// }

// // IsInitialized returns whether the system is initialized
// func (fs *FacialSystem) IsInitialized() bool {
// 	fs.mu.Lock()
// 	defer fs.mu.Unlock()

// 	return fs.isInitialized
// }

// // GetRegisteredUserCount returns the number of registered users
// func (fs *FacialSystem) GetRegisteredUserCount() (int, error) {
// 	// Get all users from database
// 	users, err := dbops.GetAllData("users", nil)
// 	if err != nil {
// 		return 0, fmt.Errorf("failed to get users: %v", err)
// 	}

// 	return len(users), nil
// }
