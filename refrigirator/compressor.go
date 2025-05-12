package refrigirator

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// Constants for our compression format
const (
	MAGIC_HEADER      = uint32(0x4B424500) // "KBE\0" in hex
	FORMAT_VERSION    = uint8(1)           // Version of the format
	COEFF_PRECISION   = 10                 // Number of bits for fractional part of coefficient
	MAX_DEGREE        = 5                  // Maximum polynomial degree
	SEPARATOR_PATTERN = uint16(0xFFFA)     // Pattern used to separate segments
)

// Point represents an (x,y) coordinate in our data
type Point struct {
	X int
	Y byte
}

// BinaryPolynomial represents a polynomial with quantized coefficients
type BinaryPolynomial struct {
	Degree       uint8   // Degree of polynomial
	Coefficients []int16 // Quantized coefficients
}

// Debug variables
var debugMode = false

// quantizeCoefficient converts a float64 coefficient to a quantized int16
func quantizeCoefficient(coeff float64) int16 {
	// Scale the coefficient to fit in int16 range with COEFF_PRECISION bits for fraction
	scaledCoeff := coeff * (1 << COEFF_PRECISION)

	// Clamp to int16 range
	if scaledCoeff > 32767 {
		return 32767
	}
	if scaledCoeff < -32768 {
		return -32768
	}

	return int16(math.Round(scaledCoeff))
}

// dequantizeCoefficient converts a quantized int16 back to float64
func dequantizeCoefficient(quantized int16) float64 {
	return float64(quantized) / (1 << COEFF_PRECISION)
}

// fitPolynomial fits a polynomial of specified degree to the given points
func fitPolynomial(points []Point, degree int) []float64 {
	// Limit degree to maximum
	if degree > MAX_DEGREE {
		degree = MAX_DEGREE
	}

	// Ensure we have enough points
	n := len(points)
	if n <= degree {
		// Not enough points, reduce degree
		degree = n - 1
		if degree < 0 {
			degree = 0
		}
	}

	// For very small segments, just store the raw values
	if degree == 0 {
		result := make([]float64, 1)
		if n > 0 {
			result[0] = float64(points[0].Y)
		}
		return result
	}

	// Extract x and y values
	x := make([]float64, n)
	y := make([]float64, n)
	for i, p := range points {
		x[i] = float64(p.X)
		y[i] = float64(p.Y)
	}

	// Use polynomial regression
	coeffs := polyfit(x, y, degree)

	return coeffs
}

// polyfit performs polynomial regression using the least squares method
func polyfit(x, y []float64, degree int) []float64 {
	n := len(x)
	if n <= 1 || degree < 1 {
		// Not enough data points or invalid degree
		coeffs := make([]float64, 1)
		if n > 0 {
			// Just use the average y value
			sum := 0.0
			for _, yi := range y {
				sum += yi
			}
			coeffs[0] = sum / float64(n)
		}
		return coeffs
	}

	// Normalize x values to improve numerical stability
	xmin, xmax := x[0], x[0]
	for _, xi := range x {
		if xi < xmin {
			xmin = xi
		}
		if xi > xmax {
			xmax = xi
		}
	}

	// Prevent division by zero
	if xmax == xmin {
		xmax = xmin + 1
	}

	// Normalize to [-1, 1] range
	xnorm := make([]float64, n)
	for i, xi := range x {
		xnorm[i] = 2*(xi-xmin)/(xmax-xmin) - 1
	}

	// Build design matrix
	a := make([][]float64, n)
	for i := range a {
		a[i] = make([]float64, degree+1)
		a[i][0] = 1.0 // Constant term
		for j := 1; j <= degree; j++ {
			a[i][j] = a[i][j-1] * xnorm[i] // x^j term
		}
	}

	// Transpose of A
	at := make([][]float64, degree+1)
	for i := range at {
		at[i] = make([]float64, n)
		for j := 0; j < n; j++ {
			at[i][j] = a[j][i]
		}
	}

	// A^T * A
	ata := make([][]float64, degree+1)
	for i := range ata {
		ata[i] = make([]float64, degree+1)
		for j := 0; j <= degree; j++ {
			for k := 0; k < n; k++ {
				ata[i][j] += at[i][k] * a[k][j]
			}
		}
	}

	// A^T * y
	aty := make([]float64, degree+1)
	for i := range aty {
		for j := 0; j < n; j++ {
			aty[i] += at[i][j] * y[j]
		}
	}

	// Solve (A^T * A) * coeffs = (A^T * y) using Gaussian elimination
	// Add small values to diagonal for numerical stability
	for i := 0; i <= degree; i++ {
		ata[i][i] += 1e-10
	}

	// Forward elimination
	for i := 0; i <= degree; i++ {
		// Find pivot
		pivotRow := i
		for k := i + 1; k <= degree; k++ {
			if math.Abs(ata[k][i]) > math.Abs(ata[pivotRow][i]) {
				pivotRow = k
			}
		}

		// Swap rows if needed
		if pivotRow != i {
			ata[i], ata[pivotRow] = ata[pivotRow], ata[i]
			aty[i], aty[pivotRow] = aty[pivotRow], aty[i]
		}

		// Skip if pivot is too small
		if math.Abs(ata[i][i]) < 1e-10 {
			continue
		}

		// Scale pivot row
		pivot := ata[i][i]
		for j := i; j <= degree; j++ {
			ata[i][j] /= pivot
		}
		aty[i] /= pivot

		// Eliminate below
		for k := i + 1; k <= degree; k++ {
			factor := ata[k][i]
			for j := i; j <= degree; j++ {
				ata[k][j] -= factor * ata[i][j]
			}
			aty[k] -= factor * aty[i]
		}
	}

	// Back substitution
	coeffs := make([]float64, degree+1)
	for i := degree; i >= 0; i-- {
		coeffs[i] = aty[i]
		for j := i + 1; j <= degree; j++ {
			coeffs[i] -= ata[i][j] * coeffs[j]
		}
	}

	// Convert coefficients back to original x scale
	result := make([]float64, degree+1)
	// This transformation is more complex because we need to expand (2*(x-xmin)/(xmax-xmin) - 1)^j
	// We'll use a simpler approach and just evaluate at original x points
	for i := 0; i < n; i++ {
		y_pred := 0.0
		for j := 0; j <= degree; j++ {
			y_pred += coeffs[j] * math.Pow(xnorm[i], float64(j))
		}

		// Use the difference to adjust the constant term
		if i == 0 {
			result[0] = y[i] - (y_pred - coeffs[0])
		}
	}

	// Copy the higher degree coefficients directly
	// Note: This is a simplification
	for j := 1; j <= degree; j++ {
		result[j] = coeffs[j] * math.Pow(2/(xmax-xmin), float64(j))
	}

	return result
}

// quantizePolynomial converts float coefficients to quantized binary format
func quantizePolynomial(coeffs []float64) BinaryPolynomial {
	degree := uint8(len(coeffs) - 1)
	if degree > MAX_DEGREE {
		degree = MAX_DEGREE
	}

	quantized := make([]int16, degree+1)
	for i := 0; i <= int(degree); i++ {
		quantized[i] = quantizeCoefficient(coeffs[i])
	}

	return BinaryPolynomial{
		Degree:       degree,
		Coefficients: quantized,
	}
}

// evaluatePolynomial evaluates a polynomial at a given x value
func evaluatePolynomial(coeffs []float64, x float64) float64 {
	result := 0.0
	xPower := 1.0

	for _, coeff := range coeffs {
		result += coeff * xPower
		xPower *= x
	}

	return result
}

// evaluateBinaryPolynomial evaluates a binary polynomial at a given x value
func evaluateBinaryPolynomial(poly BinaryPolynomial, x float64) float64 {
	result := 0.0
	xPower := 1.0

	for i := 0; i <= int(poly.Degree); i++ {
		coeff := dequantizeCoefficient(poly.Coefficients[i])
		result += coeff * xPower
		xPower *= x
	}

	return result
}

// writeBinaryPolynomial writes a binary polynomial to a byte buffer
func writeBinaryPolynomial(buf *bytes.Buffer, poly BinaryPolynomial) error {
	// Write degree (1 byte)
	err := binary.Write(buf, binary.LittleEndian, poly.Degree)
	if err != nil {
		return err
	}

	// Write coefficients (2 bytes each)
	for i := 0; i <= int(poly.Degree); i++ {
		err = binary.Write(buf, binary.LittleEndian, poly.Coefficients[i])
		if err != nil {
			return err
		}
	}

	// Write separator
	err = binary.Write(buf, binary.LittleEndian, SEPARATOR_PATTERN)
	if err != nil {
		return err
	}

	return nil
}

// readBinaryPolynomial reads a binary polynomial from a byte buffer
func readBinaryPolynomial(buf *bytes.Buffer) (BinaryPolynomial, error) {
	var degree uint8

	// Read degree
	err := binary.Read(buf, binary.LittleEndian, &degree)
	if err != nil {
		return BinaryPolynomial{}, fmt.Errorf("failed to read degree: %w", err)
	}

	if degree > MAX_DEGREE {
		return BinaryPolynomial{}, fmt.Errorf("invalid polynomial degree: %d", degree)
	}

	// Read coefficients
	coeffs := make([]int16, degree+1)
	for i := 0; i <= int(degree); i++ {
		err = binary.Read(buf, binary.LittleEndian, &coeffs[i])
		if err != nil {
			return BinaryPolynomial{}, fmt.Errorf("failed to read coefficient %d: %w", i, err)
		}
	}

	// Read and verify separator
	var separator uint16
	err = binary.Read(buf, binary.LittleEndian, &separator)
	if err != nil {
		return BinaryPolynomial{}, fmt.Errorf("failed to read separator: %w", err)
	}

	if separator != SEPARATOR_PATTERN {
		return BinaryPolynomial{}, fmt.Errorf("invalid separator pattern: %x, expected %x", separator, SEPARATOR_PATTERN)
	}

	return BinaryPolynomial{
		Degree:       degree,
		Coefficients: coeffs,
	}, nil
}

// calculateOptimalSegmentSize determines the best segment size
func calculateOptimalSegmentSize(fileSize int) int {
	// For small files (< 100KB), use smaller segments for better compression
	if fileSize < 100*1024 {
		return 32
	}

	// For small-medium files (< 1MB), use small segments
	if fileSize < 1024*1024 {
		return 64
	}

	// For medium files (1MB - 10MB), use medium segments
	if fileSize < 10*1024*1024 {
		return 128
	}

	// For large files (> 10MB), use larger segments
	return 256
}

// maxGoroutines determines the maximum number of goroutines to use
func maxGoroutines() int {
	numCPU := runtime.NumCPU()
	// Use at most numCPU goroutines, with a minimum of 2 and maximum of 16
	return int(math.Max(2, math.Min(float64(numCPU), 16)))
}

// compressFile compresses the input file using polynomial interpolation
func compressFile(inputPath, outputPath string) error {
	// Read input file
	data, err := ioutil.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	// Determine optimal segment size
	segmentSize := calculateOptimalSegmentSize(len(data))
	maxConcurrent := maxGoroutines()

	fmt.Printf("Using segment size: %d and maximum %d concurrent goroutines\n",
		segmentSize, maxConcurrent)

	// Create points from data (x is the position, y is the byte value)
	points := make([]Point, len(data))
	for i, b := range data {
		points[i] = Point{X: i + 1, Y: b} // X starts from 1 for simplicity
	}

	// Process the file in segments to handle large files
	numPoints := len(points)
	numSegments := (numPoints + segmentSize - 1) / segmentSize

	// Initialize result storage
	binaryPolynomials := make([]BinaryPolynomial, numSegments)

	// Use a worker pool to limit concurrency
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxConcurrent)

	fmt.Printf("Processing %d segments...\n", numSegments)
	startTime := time.Now()

	// Process segments
	for i := 0; i < numSegments; i++ {
		segmentStart := i * segmentSize
		segmentEnd := (i + 1) * segmentSize
		if segmentEnd > numPoints {
			segmentEnd = numPoints
		}

		segment := points[segmentStart:segmentEnd]

		wg.Add(1)
		semaphore <- struct{}{} // Acquire semaphore

		go func(idx int, segment []Point) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release semaphore

			// Determine appropriate polynomial degree
			segLen := len(segment)
			var degree int

			// Adjust degree based on segment size
			if segLen <= 2 {
				degree = 1 // Linear for tiny segments
			} else if segLen <= 5 {
				degree = 2 // Quadratic for small segments
			} else if segLen <= 10 {
				degree = 3 // Cubic for medium segments
			} else {
				// For larger segments, use higher degree but cap it
				degree = int(math.Min(4, float64(segLen/20)))
			}

			// Fit polynomial
			coeffs := fitPolynomial(segment, degree)

			// Convert to binary format
			binaryPolynomials[idx] = quantizePolynomial(coeffs)

			if idx%100 == 0 {
				fmt.Printf("Processed segment %d/%d\n", idx, numSegments)
			}
		}(i, segment)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	processingTime := time.Since(startTime)
	fmt.Printf("Processed all segments in %v\n", processingTime)

	// Create a buffer to hold all the data
	buf := new(bytes.Buffer)

	// Write magic header
	err = binary.Write(buf, binary.LittleEndian, MAGIC_HEADER)
	if err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write format version
	err = binary.Write(buf, binary.LittleEndian, FORMAT_VERSION)
	if err != nil {
		return fmt.Errorf("failed to write format version: %w", err)
	}

	// Write metadata
	// [SegmentSize(4 bytes)][NumSegments(4 bytes)][TotalSize(4 bytes)]
	err = binary.Write(buf, binary.LittleEndian, uint32(segmentSize))
	if err != nil {
		return fmt.Errorf("failed to write segment size: %w", err)
	}

	err = binary.Write(buf, binary.LittleEndian, uint32(numSegments))
	if err != nil {
		return fmt.Errorf("failed to write number of segments: %w", err)
	}

	err = binary.Write(buf, binary.LittleEndian, uint32(numPoints))
	if err != nil {
		return fmt.Errorf("failed to write total size: %w", err)
	}

	// Write polynomials
	for i, poly := range binaryPolynomials {
		err = writeBinaryPolynomial(buf, poly)
		if err != nil {
			return fmt.Errorf("failed to write polynomial %d: %w", i, err)
		}
	}

	// Write the buffer to the output file
	err = ioutil.WriteFile(outputPath, buf.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	// Calculate and display compression metrics
	originalSize := len(data)
	compressedSize := buf.Len()
	compressionRatio := float64(originalSize) / float64(compressedSize)

	fmt.Printf("Original size: %d bytes\n", originalSize)
	fmt.Printf("Compressed size: %d bytes\n", compressedSize)
	fmt.Printf("Compression ratio: %.2f:1\n", compressionRatio)

	return nil
}

// verifyCompression checks if the decompressed data matches the original data
func verifyCompression(originalPath, decompressedPath string) (bool, error) {
	// Read original file
	originalData, err := ioutil.ReadFile(originalPath)
	if err != nil {
		return false, fmt.Errorf("failed to read original file: %w", err)
	}

	// Read decompressed file
	decompressedData, err := ioutil.ReadFile(decompressedPath)
	if err != nil {
		return false, fmt.Errorf("failed to read decompressed file: %w", err)
	}

	// Check if the files have the same length
	if len(originalData) != len(decompressedData) {
		fmt.Printf("Size mismatch: original %d bytes, decompressed %d bytes\n",
			len(originalData), len(decompressedData))
		return false, nil
	}

	// Count differences
	differences := 0
	for i := 0; i < len(originalData); i++ {
		if originalData[i] != decompressedData[i] {
			differences++
		}
	}

	if differences > 0 {
		fmt.Printf("Found %d differences out of %d bytes (%.2f%%)\n",
			differences, len(originalData),
			100.0*float64(differences)/float64(len(originalData)))
		return false, nil
	}

	return true, nil
}

// benchmarkCompression measures the performance of compression and decompression
func benchmarkCompression(filePath string) error {
	// Generate a temporary file path
	tempDir, err := ioutil.TempDir("", "compress-benchmark")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	compressedPath := filepath.Join(tempDir, "compressed.kbe")
	decompressedPath := filepath.Join(tempDir, "decompressed")

	// Measure compression time
	fmt.Println("Benchmarking compression...")
	startCompress := time.Now()
	err = compressFile(filePath, compressedPath)
	if err != nil {
		return fmt.Errorf("compression error: %w", err)
	}
	compressTime := time.Since(startCompress)

	// Get file sizes
	originalInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat original file: %w", err)
	}

	compressedInfo, err := os.Stat(compressedPath)
	if err != nil {
		return fmt.Errorf("failed to stat compressed file: %w", err)
	}

	// Calculate compression ratio
	originalSize := originalInfo.Size()
	compressedSize := compressedInfo.Size()
	compressionRatio := float64(originalSize) / float64(compressedSize)

	// Measure decompression time
	fmt.Println("Benchmarking decompression...")
	startDecompress := time.Now()
	err = decompressFile(compressedPath, decompressedPath)
	if err != nil {
		return fmt.Errorf("decompression error: %w", err)
	}
	decompressTime := time.Since(startDecompress)

	// Verify correctness
	identical, err := verifyCompression(filePath, decompressedPath)
	if err != nil {
		return fmt.Errorf("verification error: %w", err)
	}

	// Print results
	fmt.Printf("Benchmark results for %s:\n", filePath)
	fmt.Printf("Original size: %d bytes\n", originalSize)
	fmt.Printf("Compressed size: %d bytes\n", compressedSize)
	fmt.Printf("Compression ratio: %.2f:1\n", compressionRatio)
	fmt.Printf("Compression time: %v\n", compressTime)
	fmt.Printf("Decompression time: %v\n", decompressTime)
	fmt.Printf("Verification: %v\n", identical)

	return nil
}

// verifyFileIntegrity checks if a compressed file has valid structure without decompressing it
func verifyFileIntegrity(filePath string) (bool, error) {
	// Read compressed file
	compressedData, err := ioutil.ReadFile(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to read input file: %w", err)
	}

	// Create a buffer to read from
	buf := bytes.NewBuffer(compressedData)

	// Check file size
	if buf.Len() < 17 { // Magic(4) + Version(1) + SegSize(4) + NumSegs(4) + TotalSize(4)
		return false, fmt.Errorf("file too small to be a valid .kbe file")
	}

	// Read magic header
	var magic uint32
	err = binary.Read(buf, binary.LittleEndian, &magic)
	if err != nil {
		return false, fmt.Errorf("failed to read header: %w", err)
	}

	if magic != MAGIC_HEADER {
		return false, fmt.Errorf("invalid file format, expected magic header %X, got %X", MAGIC_HEADER, magic)
	}

	// Read format version
	var version uint8
	err = binary.Read(buf, binary.LittleEndian, &version)
	if err != nil {
		return false, fmt.Errorf("failed to read format version: %w", err)
	}

	if version != FORMAT_VERSION {
		return false, fmt.Errorf("unsupported format version: %d, expected %d", version, FORMAT_VERSION)
	}

	// Read metadata
	var segmentSize, numSegments, totalSize uint32

	err = binary.Read(buf, binary.LittleEndian, &segmentSize)
	if err != nil {
		return false, fmt.Errorf("failed to read segment size: %w", err)
	}

	err = binary.Read(buf, binary.LittleEndian, &numSegments)
	if err != nil {
		return false, fmt.Errorf("failed to read number of segments: %w", err)
	}

	err = binary.Read(buf, binary.LittleEndian, &totalSize)
	if err != nil {
		return false, fmt.Errorf("failed to read total size: %w", err)
	}

	fmt.Printf("File integrity check: %d segments, segment size %d, total size %d bytes\n",
		numSegments, segmentSize, totalSize)

	// Verify that we have enough data for all the polynomials
	// Each polynomial requires at least: 1 byte (degree) + 2 bytes (min coeff) + 2 bytes (separator)
	minExpectedSize := 17 + (numSegments * 5)
	if int64(buf.Len()+17) < int64(minExpectedSize) {
		return false, fmt.Errorf("file too small to contain all polynomials")
	}

	// Read polynomials to verify integrity
	for i := 0; i < int(numSegments); i++ {
		_, err := readBinaryPolynomial(buf)
		if err != nil {
			return false, fmt.Errorf("corrupt polynomial at segment %d: %w", i, err)
		}
	}

	// All checks passed
	return true, nil
}

// Updated decompressFile with enhanced error handling
func decompressFile(inputPath, outputPath string) error {
	// First verify file integrity
	valid, err := verifyFileIntegrity(inputPath)
	if err != nil {
		return fmt.Errorf("file integrity check failed: %w", err)
	}

	if !valid {
		return fmt.Errorf("file integrity check failed with unknown error")
	}

	fmt.Println("File integrity check passed, proceeding with decompression")

	// Read compressed file
	compressedData, err := ioutil.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	// Create a buffer to read from
	buf := bytes.NewBuffer(compressedData)

	// Skip the header and metadata that we've already verified
	// Magic(4) + Version(1) + SegSize(4) + NumSegs(4) + TotalSize(4)
	_, err = buf.Read(make([]byte, 17))
	if err != nil {
		return fmt.Errorf("failed to skip header: %w", err)
	}

	// Read metadata again (we need the values)
	var segmentSize, numSegments, totalSize uint32
	segmentSize = binary.LittleEndian.Uint32(compressedData[5:9])
	numSegments = binary.LittleEndian.Uint32(compressedData[9:13])
	totalSize = binary.LittleEndian.Uint32(compressedData[13:17])

	fmt.Printf("Decompressing file with %d segments, segment size %d, total size %d bytes\n",
		numSegments, segmentSize, totalSize)

	// Read polynomials with explicit error handling
	polynomials := make([]BinaryPolynomial, numSegments)
	for i := 0; i < int(numSegments); i++ {
		poly, err := readBinaryPolynomial(buf)
		if err != nil {
			return fmt.Errorf("error reading polynomial %d: %w", i, err)
		}
		polynomials[i] = poly

		if debugMode && i < 5 {
			fmt.Printf("Polynomial %d: degree %d, coeffs: %v\n", i, poly.Degree, poly.Coefficients)
		}
	}

	// Reconstruct the original data
	decompressedData := make([]byte, totalSize)

	// Use a worker pool to limit concurrency
	var wg sync.WaitGroup
	maxConcurrent := maxGoroutines()
	semaphore := make(chan struct{}, maxConcurrent)
	var mutex sync.Mutex
	errChan := make(chan error, numSegments) // Channel to collect errors

	fmt.Printf("Using maximum %d concurrent goroutines for decompression\n", maxConcurrent)
	startTime := time.Now()

	// Process each segment
	for i := 0; i < int(numSegments); i++ {
		wg.Add(1)
		semaphore <- struct{}{} // Acquire semaphore

		go func(idx int, poly BinaryPolynomial) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release semaphore

			// Recover from any panics
			defer func() {
				if r := recover(); r != nil {
					errChan <- fmt.Errorf("panic in segment %d: %v", idx, r)
				}
			}()

			start := idx*int(segmentSize) + 1 // +1 because x starts from 1
			end := (idx + 1) * int(segmentSize)
			if end > int(totalSize) {
				end = int(totalSize)
			}

			// Validate segment bounds
			if start > int(totalSize) || start < 1 {
				errChan <- fmt.Errorf("invalid segment start index: %d", start)
				return
			}

			// Reconstruct segment data
			segmentData := make([]byte, end-start+1)
			for j := 0; j < end-start+1; j++ {
				x := float64(start + j)
				y := evaluateBinaryPolynomial(poly, x)

				// Check for NaN or infinity
				if math.IsNaN(y) || math.IsInf(y, 0) {
					errChan <- fmt.Errorf("polynomial evaluation resulted in invalid value at segment %d, x=%f", idx, x)
					return
				}

				// Clamp to valid byte range and convert back to byte
				segmentData[j] = byte(math.Max(0, math.Min(255, math.Round(y))))
			}

			mutex.Lock()
			copy(decompressedData[start-1:end], segmentData)
			mutex.Unlock()

			if idx%100 == 0 {
				fmt.Printf("Decompressed segment %d/%d\n", idx, int(numSegments))
			}
		}(i, polynomials[i])
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	decompressTime := time.Since(startTime)
	fmt.Printf("Decompressed all segments in %v\n", decompressTime)

	// Write decompressed data to output file
	err = ioutil.WriteFile(outputPath, decompressedData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}

type Refrigirator struct {
	version    string
	indexCache any
}

func CreateRefrigirator(version string, indexdb any) *Refrigirator {
	return &Refrigirator{
		version:    version,
		indexCache: indexdb,
	}
}

func (RG *Refrigirator) Refrigirate(mode string) {
	// Parse command-line flags
	modePtr := flag.String("mode", "", "Operation mode: 'compress', 'decompress', 'check', 'verify', or 'benchmark'")
	inputPtr := flag.String("input", "", "Input file path")
	outputPtr := flag.String("output", "", "Output file path (optional, will be determined automatically if not provided)")
	//segmentSizePtr := flag.Int("segment-size", 0, "Override the automatic segment size (optional)")
	debugPtr := flag.Bool("debug", false, "Enable debug mode")
	flag.Parse()

	debugMode = *debugPtr

	// Validate inputs
	if *modePtr != "compress" && *modePtr != "decompress" && *modePtr != "check" && *modePtr != "verify" && *modePtr != "benchmark" {
		fmt.Println("Error: mode must be 'compress', 'decompress', 'check', 'verify', or 'benchmark'")
		flag.Usage()
		os.Exit(1)
	}

	if *inputPtr == "" {
		fmt.Println("Error: input file path must be provided")
		flag.Usage()
		os.Exit(1)
	}

	// Determine output file path if not provided
	outputPath := *outputPtr
	if outputPath == "" {
		if *modePtr == "compress" {
			outputPath = *inputPtr + ".kbe"
		} else if *modePtr == "decompress" {
			if filepath.Ext(*inputPtr) == ".kbe" {
				outputPath = (*inputPtr)[:len(*inputPtr)-4] // Remove .kbe extension
			} else {
				outputPath = *inputPtr + ".decompressed" // Fallback if input doesn't have .kbe extension
			}
		}
	}

	// Perform the operation
	if *modePtr == "compress" {
		err := compressFile(*inputPtr, outputPath)
		if err != nil {
			fmt.Printf("Compression error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully compressed %s to %s\n", *inputPtr, outputPath)
	} else if *modePtr == "decompress" {
		err := decompressFile(*inputPtr, outputPath)
		if err != nil {
			fmt.Printf("Decompression error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully decompressed %s to %s\n", *inputPtr, outputPath)
	} else if *modePtr == "check" {
		// Just verify file integrity without decompressing
		valid, err := verifyFileIntegrity(*inputPtr)
		if err != nil {
			fmt.Printf("File integrity check failed: %v\n", err)
			os.Exit(1)
		}
		if valid {
			fmt.Printf("File %s passed integrity check\n", *inputPtr)
		} else {
			fmt.Printf("File %s failed integrity check\n", *inputPtr)
			os.Exit(1)
		}
	} else if *modePtr == "verify" {
		originalPath := *inputPtr
		compressedPath := originalPath + ".kbe"
		decompressedPath := originalPath + ".decompressed"
		if *outputPtr != "" {
			decompressedPath = *outputPtr
		}

		// Compress the original file
		fmt.Println("Compressing file for verification...")
		err := compressFile(originalPath, compressedPath)
		if err != nil {
			fmt.Printf("Compression error: %v\n", err)
			os.Exit(1)
		}

		// Decompress the compressed file
		fmt.Println("Decompressing file for verification...")
		err = decompressFile(compressedPath, decompressedPath)
		if err != nil {
			fmt.Printf("Decompression error: %v\n", err)
			os.Exit(1)
		}

		// Verify the results
		fmt.Println("Comparing original and decompressed files...")
		identical, err := verifyCompression(originalPath, decompressedPath)
		if err != nil {
			fmt.Printf("Verification error: %v\n", err)
			os.Exit(1)
		}

		if identical {
			fmt.Println("Verification successful: Original and decompressed files are identical")
		} else {
			fmt.Println("Verification failed: Original and decompressed files differ")
			os.Exit(1)
		}
	} else if *modePtr == "benchmark" {
		err := benchmarkCompression(*inputPtr)
		if err != nil {
			fmt.Printf("Benchmark error: %v\n", err)
			os.Exit(1)
		}
	}
}
