// package security

// import (
// 	"fmt"
// 	"os"
// 	"os/signal"
// 	"time"

// 	"github.com/Kagami/go-face"
// 	"gocv.io/x/gocv"
// )

// const (
// 	// Path to the directory with the model data
// 	modelDir = "./models"
// 	// Threshold for face recognition confidence
// 	recognitionThreshold = 0.6
// 	// Checking interval in seconds
// 	checkInterval = 30
// 	// Sample size to train the model
// 	sampleSize = 5
// )

// type FaceRecognitionSystem struct {
// 	recognizer      *face.Recognizer
// 	samples         []face.Descriptor
// 	camera          *gocv.VideoCapture
// 	trainedPersonID int32
// 	isLocked        bool
// }

// func NewFaceRecognitionSystem() (*FaceRecognitionSystem, error) {
// 	// Initialize face recognizer
// 	rec, err := face.NewRecognizer(modelDir)
// 	if err != nil {
// 		return nil, fmt.Errorf("cannot initialize recognizer: %v", err)
// 	}

// 	// Initialize camera
// 	cam, err := gocv.OpenVideoCapture(0)
// 	if err != nil {
// 		rec.Close()
// 		return nil, fmt.Errorf("cannot open camera: %v", err)
// 	}

// 	return &FaceRecognitionSystem{
// 		recognizer:      rec,
// 		camera:          cam,
// 		trainedPersonID: 0,
// 		isLocked:        false,
// 	}, nil
// }

// func (frs *FaceRecognitionSystem) Close() {
// 	if frs.camera != nil {
// 		frs.camera.Close()
// 	}
// 	if frs.recognizer != nil {
// 		frs.recognizer.Close()
// 	}
// }

// func (frs *FaceRecognitionSystem) TrainModel() error {
// 	fmt.Println("Training model. Please sit in front of the camera.")
// 	time.Sleep(3 * time.Second) // Give time for the user to prepare

// 	var samples []face.Descriptor

// 	// Capture multiple samples for better recognition
// 	for i := 0; i < sampleSize; i++ {
// 		fmt.Printf("Capturing sample %d/%d...\n", i+1, sampleSize)

// 		// Capture frame
// 		img := gocv.NewMat()
// 		if ok := frs.camera.Read(&img); !ok {
// 			img.Close()
// 			return fmt.Errorf("cannot read from camera")
// 		}

// 		// Save frame temporarily
// 		tempFile := fmt.Sprintf("temp_sample_%d.jpg", i)
// 		if ok := gocv.IMWrite(tempFile, img); !ok {
// 			img.Close()
// 			return fmt.Errorf("failed to save image")
// 		}
// 		img.Close()

// 		// Recognize faces in the image
// 		faces, err := frs.recognizer.RecognizeFile(tempFile)
// 		os.Remove(tempFile) // Clean up temp file

// 		if err != nil {
// 			return fmt.Errorf("recognition error: %v", err)
// 		}

// 		if len(faces) == 0 {
// 			fmt.Println("No face detected. Please make sure your face is visible.")
// 			i-- // Retry this sample
// 			time.Sleep(1 * time.Second)
// 			continue
// 		}

// 		if len(faces) > 1 {
// 			fmt.Println("Multiple faces detected. Please ensure only one person is in view.")
// 			i-- // Retry this sample
// 			time.Sleep(1 * time.Second)
// 			continue
// 		}

// 		// Add face descriptor to samples
// 		samples = append(samples, faces[0].Descriptor)
// 		time.Sleep(500 * time.Millisecond) // Small delay between captures
// 	}

// 	if len(samples) == 0 {
// 		return fmt.Errorf("couldn't capture any valid face samples")
// 	}

// 	frs.samples = samples
// 	fmt.Println("Model successfully trained!")
// 	return nil
// }

// func (frs *FaceRecognitionSystem) CheckFace() bool {
// 	// Capture frame
// 	img := gocv.NewMat()
// 	defer img.Close()

// 	if ok := frs.camera.Read(&img); !ok {
// 		fmt.Println("Cannot read from camera")
// 		return false
// 	}

// 	// Save frame temporarily
// 	tempFile := "temp_check.jpg"
// 	if ok := gocv.IMWrite(tempFile, img); !ok {
// 		fmt.Println("Failed to save image")
// 		return false
// 	}

// 	// Recognize faces in the image
// 	faces, err := frs.recognizer.RecognizeFile(tempFile)
// 	os.Remove(tempFile) // Clean up temp file

// 	if err != nil {
// 		fmt.Printf("Recognition error: %v\n", err)
// 		return false
// 	}

// 	if len(faces) == 0 {
// 		return false // No face detected
// 	}

// 	// Check if the detected face matches the trained face
// 	for _, sample := range frs.samples {
// 		// Check similarity with each sample
// 		for _, face := range faces {
// 			// Calculate Euclidean distance between face descriptors
// 			// Lower distance means more similarity
// 			dist := euclideanDistance(sample, face.Descriptor)
// 			if dist < recognitionThreshold {
// 				return true // Face matches
// 			}
// 		}
// 	}

// 	return false // No matching face
// }

// // Calculate Euclidean distance between two face descriptors
// func euclideanDistance(a, b face.Descriptor) float32 {
// 	var sum float32
// 	for i := 0; i < len(a); i++ {
// 		diff := a[i] - b[i]
// 		sum += diff * diff
// 	}
// 	return sum
// }

// func (frs *FaceRecognitionSystem) StartMonitoring() {
// 	ticker := time.NewTicker(checkInterval * time.Second)
// 	defer ticker.Stop()

// 	// Set up graceful shutdown
// 	c := make(chan os.Signal, 1)
// 	signal.Notify(c, os.Interrupt)

// 	fmt.Println("Monitoring started. Press Ctrl+C to stop.")

// 	// Initial check
// 	go frs.performCheck()

// 	for {
// 		select {
// 		case <-ticker.C:
// 			go frs.performCheck()
// 		case <-c:
// 			fmt.Println("\nShutting down...")
// 			return
// 		}
// 	}
// }

// func (frs *FaceRecognitionSystem) performCheck() {
// 	authorized := frs.CheckFace()

// 	if !authorized {
// 		if !frs.isLocked {
// 			fmt.Println("Locking")
// 			frs.isLocked = true
// 		}
// 	} else {
// 		if frs.isLocked {
// 			fmt.Println("All good")
// 			frs.isLocked = false
// 		} else {
// 			fmt.Println("All good")
// 		}
// 	}
// }

// func main() {
// 	// Check if model directory exists
// 	if _, err := os.Stat(modelDir); os.IsNotExist(err) {
// 		fmt.Printf("Model directory '%s' does not exist. Please create it and download the required models.\n", modelDir)
// 		return
// 	}

// 	frs, err := NewFaceRecognitionSystem()
// 	if err != nil {
// 		fmt.Printf("Error initializing face recognition system: %v\n", err)
// 		return
// 	}
// 	defer frs.Close()

// 	fmt.Println("Face Recognition System initialized.")

// 	// Train the model with current user's face
// 	if err := frs.TrainModel(); err != nil {
// 		fmt.Printf("Training error: %v\n", err)
// 		return
// 	}

// 	// Start periodic monitoring
// 	frs.StartMonitoring()
// }
