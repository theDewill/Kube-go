package refrigirator

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	ChunkSize   = 8192   // Process data in 8KB chunks
	MaxDegree   = 6      // Maximum polynomial degree
	HeaderMagic = "KBE1" // File format identifier
)

func Compressor(modePtr string, inputDir string, outputDir string) error {

	// Validate arguments
	if modePtr != "compress" && modePtr != "decompress" {
		fmt.Println("Error: mode must be either 'compress' or 'decompress'")
		os.Exit(1)
	}

	if inputDir == "" {
		fmt.Println("Error: input file path is required")
		os.Exit(1)
	}

	startTime := time.Now()
	var rferr error

	// Execute the appropriate mode
	if modePtr == "compress" {
		rferr = compressFile(inputDir, outputDir)
		if rferr != nil {
			fmt.Printf("Compression error: %v\n", rferr)
			os.Exit(1)
			return rferr

		}
	} else {
		rferr = decompressFile(inputDir, outputDir)
		if rferr != nil {
			fmt.Printf("Decompression error: %v\n", rferr)
			os.Exit(1)
			return rferr
		}
	}

	elapsedTime := time.Since(startTime)
	fmt.Printf("Operation completed successfully in %v\n", elapsedTime)
	return nil
}

// compressFile compresses a file using our improved approach
func compressFile(inputPath string, outputDir string) error {
	// Open input file
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer inputFile.Close()

	// Get file info for size
	fileInfo, err := inputFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}
	fileSize := fileInfo.Size()

	// Create output file with .kbe extension
	//outputPath := inputPath + ".kbe"
	var outputPath string
	if outputDir == "" {
		outputPath = inputPath + ".kbe"
	} else {
		fileName := filepath.Base(inputPath) + ".kbe"
		outputPath = filepath.Join(outputDir, fileName)

		// Create output directory if it doesn't exist
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	// Create buffered writer for better performance
	bufWriter := bufio.NewWriter(outputFile)
	defer bufWriter.Flush()

	// Write header
	// Magic bytes
	bufWriter.WriteString(HeaderMagic)

	// File size
	binary.Write(bufWriter, binary.LittleEndian, fileSize)

	// Number of chunks
	numChunks := (fileSize + ChunkSize - 1) / ChunkSize
	binary.Write(bufWriter, binary.LittleEndian, int32(numChunks))

	fmt.Printf("Processing %s in %d chunks...\n", inputPath, numChunks)

	// Process each chunk sequentially to avoid deadlocks
	for chunkID := int64(0); chunkID < numChunks; chunkID++ {
		// Calculate chunk offset and size
		offset := chunkID * ChunkSize
		size := ChunkSize
		if offset+int64(size) > fileSize {
			size = int(fileSize - offset)
		}

		// Create buffer and read chunk
		buffer := make([]byte, size)
		_, err := inputFile.ReadAt(buffer, offset)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read chunk %d: %w", chunkID, err)
		}

		// Compress the chunk using multiple methods and pick the best one
		compressedData, compressionType, err := getBestCompression(buffer)
		if err != nil {
			return fmt.Errorf("failed to compress chunk %d: %w", chunkID, err)
		}

		// Write compression type and data length
		err = bufWriter.WriteByte(compressionType)
		if err != nil {
			return fmt.Errorf("failed to write compression type: %w", err)
		}

		dataLen := uint32(len(compressedData))
		err = binary.Write(bufWriter, binary.LittleEndian, dataLen)
		if err != nil {
			return fmt.Errorf("failed to write data length: %w", err)
		}

		// Write compressed data
		_, err = bufWriter.Write(compressedData)
		if err != nil {
			return fmt.Errorf("failed to write compressed data: %w", err)
		}

		// Show progress every 10% or so
		if chunkID%(numChunks/10+1) == 0 {
			fmt.Printf("Progress: %.1f%% (chunk %d of %d)\n",
				float64(chunkID)/float64(numChunks)*100, chunkID+1, numChunks)
		}
	}

	// Flush any remaining data
	bufWriter.Flush()

	// Calculate and display compression stats
	outputInfo, err := os.Stat(outputPath)
	if err != nil {
		return fmt.Errorf("failed to get output file info: %w", err)
	}

	outputSize := outputInfo.Size()
	ratio := float64(fileSize) / float64(outputSize)

	fmt.Printf("Compressed %s (%.2f KB) to %s (%.2f KB) with ratio %.2f:1\n",
		inputPath, float64(fileSize)/1024,
		outputPath, float64(outputSize)/1024,
		ratio)

	return nil
}

// Compression types
const (
	CompTypeRaw      byte = 0 // Raw uncompressed data
	CompTypeZLib     byte = 1 // ZLib compression
	CompTypePoly     byte = 2 // Polynomial compression
	CompTypePolyZLib byte = 3 // Polynomial + ZLib
)

// getBestCompression tries multiple compression methods and returns the best one
func getBestCompression(data []byte) ([]byte, byte, error) {
	// For very small chunks, just store raw
	if len(data) < 64 {
		return data, CompTypeRaw, nil
	}

	// Try different compression methods

	// 1. Direct zlib
	var zlibBuf bytes.Buffer
	zWriter, err := zlib.NewWriterLevel(&zlibBuf, zlib.BestCompression)
	if err != nil {
		return nil, 0, err
	}

	_, err = zWriter.Write(data)
	if err != nil {
		return nil, 0, err
	}
	zWriter.Close()
	zlibData := zlibBuf.Bytes()

	// Skip polynomial compression for larger chunks to avoid slow performance
	if len(data) > 1024 {
		// Just choose between raw and zlib
		if len(zlibData) < len(data) {
			return zlibData, CompTypeZLib, nil
		}
		return data, CompTypeRaw, nil
	}

	// 2. Try polynomial compression
	polyData, err := compressWithPolynomial(data)
	if err != nil {
		// If polynomial compression fails, fall back to zlib
		if len(zlibData) < len(data) {
			return zlibData, CompTypeZLib, nil
		}
		return data, CompTypeRaw, nil
	}

	// 3. Try polynomial + zlib
	var polyZlibBuf bytes.Buffer
	pzWriter, err := zlib.NewWriterLevel(&polyZlibBuf, zlib.BestCompression)
	if err != nil {
		return nil, 0, err
	}

	_, err = pzWriter.Write(polyData)
	if err != nil {
		return nil, 0, err
	}
	pzWriter.Close()
	polyZlibData := polyZlibBuf.Bytes()

	// Choose the best method
	candidates := []struct {
		data []byte
		typ  byte
	}{
		{data, CompTypeRaw},
		{zlibData, CompTypeZLib},
		{polyData, CompTypePoly},
		{polyZlibData, CompTypePolyZLib},
	}

	best := candidates[0]
	for _, c := range candidates[1:] {
		if len(c.data) < len(best.data) {
			best = c
		}
	}

	return best.data, best.typ, nil
}

// compressWithPolynomial uses polynomial fitting for compression
func compressWithPolynomial(data []byte) ([]byte, error) {
	// Convert bytes to points
	n := len(data)
	points := make([]float64, n)
	for i, b := range data {
		points[i] = float64(b)
	}

	// Try different polynomial degrees and pick the best one
	bestDegree := 0
	bestRMSE := math.MaxFloat64
	var bestCoeffs []float64

	// Try degrees from 1 to MaxDegree
	for degree := 1; degree <= MaxDegree; degree++ {
		if n <= degree {
			continue // Skip if not enough points
		}

		coeffs, err := fitPolynomial(points, degree)
		if err != nil {
			continue // Skip to next degree if fitting fails
		}

		// Calculate error
		rmse := calculateRMSE(points, coeffs)

		// Check if this degree is better
		if rmse < bestRMSE {
			bestRMSE = rmse
			bestDegree = degree
			bestCoeffs = coeffs
		}

		// If error is already small enough, stop trying higher degrees
		if rmse < 0.5 {
			break
		}
	}

	// If no good fit found or bestDegree is 0, return error
	if bestDegree == 0 {
		return nil, fmt.Errorf("failed to find good polynomial fit")
	}

	// Calculate residuals (errors)
	residuals := make([]byte, n)
	for i, y := range points {
		// Evaluate polynomial at x = i
		predicted := evaluatePolynomial(bestCoeffs, float64(i)/float64(n-1))

		// Calculate error and clamp to byte
		error := int(math.Round(y - predicted))
		if error < -128 {
			error = -128
		} else if error > 127 {
			error = 127
		}

		residuals[i] = byte(error + 128) // Shift to make unsigned
	}

	// Create output buffer
	var buf bytes.Buffer

	// Write degree
	buf.WriteByte(byte(bestDegree))

	// Write coefficients with reduced precision
	for _, coeff := range bestCoeffs {
		// Scale and quantize coefficients to int16
		scaled := int16(math.Round(coeff * 100))
		binary.Write(&buf, binary.LittleEndian, scaled)
	}

	// Write residuals
	buf.Write(residuals)

	return buf.Bytes(), nil
}

// fitPolynomial fits a polynomial to the given points
func fitPolynomial(points []float64, degree int) ([]float64, error) {
	n := len(points)
	if n <= degree {
		return nil, fmt.Errorf("not enough points for polynomial degree %d", degree)
	}

	// Normalize x to [0,1] for better numerical stability
	x := make([]float64, n)
	for i := range x {
		x[i] = float64(i) / float64(n-1)
	}

	// Build the Vandermonde matrix
	a := make([][]float64, n)
	for i := range a {
		a[i] = make([]float64, degree+1)
		for j := 0; j <= degree; j++ {
			a[i][j] = math.Pow(x[i], float64(j))
		}
	}

	// Build the normal equations: A^T * A * x = A^T * b
	ata := make([][]float64, degree+1)
	for i := range ata {
		ata[i] = make([]float64, degree+1)
		for j := 0; j <= degree; j++ {
			for k := 0; k < n; k++ {
				ata[i][j] += a[k][i] * a[k][j]
			}
		}
	}

	atb := make([]float64, degree+1)
	for i := range atb {
		for k := 0; k < n; k++ {
			atb[i] += a[k][i] * points[k]
		}
	}

	// Solve using Gaussian elimination with pivoting
	return solveLinearSystem(ata, atb)
}

// solveLinearSystem solves a linear system using Gaussian elimination
func solveLinearSystem(a [][]float64, b []float64) ([]float64, error) {
	n := len(b)
	augmented := make([][]float64, n)
	for i := range augmented {
		augmented[i] = make([]float64, n+1)
		for j := 0; j < n; j++ {
			augmented[i][j] = a[i][j]
		}
		augmented[i][n] = b[i]
	}

	// Forward elimination with partial pivoting
	for i := 0; i < n; i++ {
		// Find pivot
		maxRow := i
		for j := i + 1; j < n; j++ {
			if math.Abs(augmented[j][i]) > math.Abs(augmented[maxRow][i]) {
				maxRow = j
			}
		}

		// Swap rows
		augmented[i], augmented[maxRow] = augmented[maxRow], augmented[i]

		// Check for singularity
		if math.Abs(augmented[i][i]) < 1e-10 {
			return nil, fmt.Errorf("matrix is singular")
		}

		// Eliminate below
		for j := i + 1; j < n; j++ {
			factor := augmented[j][i] / augmented[i][i]
			for k := i; k <= n; k++ {
				augmented[j][k] -= factor * augmented[i][k]
			}
		}
	}

	// Back substitution
	x := make([]float64, n)
	for i := n - 1; i >= 0; i-- {
		x[i] = augmented[i][n]
		for j := i + 1; j < n; j++ {
			x[i] -= augmented[i][j] * x[j]
		}
		x[i] /= augmented[i][i]
	}

	return x, nil
}

// evaluatePolynomial evaluates a polynomial at a given x
func evaluatePolynomial(coeffs []float64, x float64) float64 {
	result := 0.0
	xPower := 1.0

	for _, coeff := range coeffs {
		result += coeff * xPower
		xPower *= x
	}

	return result
}

// calculateRMSE calculates the root mean square error between original points and polynomial
func calculateRMSE(points []float64, coeffs []float64) float64 {
	n := len(points)
	sumSquaredError := 0.0

	for i, y := range points {
		x := float64(i) / float64(n-1)
		predicted := evaluatePolynomial(coeffs, x)
		error := y - predicted
		sumSquaredError += error * error
	}

	return math.Sqrt(sumSquaredError / float64(n))
}

// decompressFile decompresses a file
func decompressFile(inputPath string, outputDir string) error {
	// Verify file extension
	if filepath.Ext(inputPath) != ".kbe" {
		return fmt.Errorf("input file must have .kbe extension for decompression")
	}

	// Open input file
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer inputFile.Close()

	// Create buffered reader
	bufReader := bufio.NewReader(inputFile)

	// Read and verify header
	magic := make([]byte, 4)
	if _, err := io.ReadFull(bufReader, magic); err != nil {
		return fmt.Errorf("failed to read magic bytes: %w", err)
	}

	if string(magic) != HeaderMagic {
		return fmt.Errorf("invalid file format: not a KBE file")
	}

	// Read original file size
	var fileSize int64
	if err := binary.Read(bufReader, binary.LittleEndian, &fileSize); err != nil {
		return fmt.Errorf("failed to read file size: %w", err)
	}

	// Read number of chunks
	var numChunks int32
	if err := binary.Read(bufReader, binary.LittleEndian, &numChunks); err != nil {
		return fmt.Errorf("failed to read number of chunks: %w", err)
	}

	// Create output file (removing .kbe extension)
	//outputPath := strings.TrimSuffix(inputPath, ".kbe")
	var outputPath string
	if outputDir == "" {
		outputPath = strings.TrimSuffix(inputPath, ".kbe")
	} else {
		fileName := strings.TrimSuffix(filepath.Base(inputPath), ".kbe")
		outputPath = filepath.Join(outputDir, fileName)

		// Create output directory if it doesn't exist
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	// Create buffered writer
	bufWriter := bufio.NewWriter(outputFile)
	defer bufWriter.Flush()

	// Process each chunk
	var bytesWritten int64
	for chunkID := int32(0); chunkID < numChunks; chunkID++ {
		// Read compression type
		compressionType, err := bufReader.ReadByte()
		if err != nil {
			return fmt.Errorf("failed to read compression type: %w", err)
		}

		// Read data length
		var dataLen uint32
		if err := binary.Read(bufReader, binary.LittleEndian, &dataLen); err != nil {
			return fmt.Errorf("failed to read data length: %w", err)
		}

		// Read compressed data
		compressedData := make([]byte, dataLen)
		if _, err := io.ReadFull(bufReader, compressedData); err != nil {
			return fmt.Errorf("failed to read compressed data: %w", err)
		}

		// Decompress based on compression type
		var decompressedData []byte

		switch compressionType {
		case CompTypeRaw:
			decompressedData = compressedData

		case CompTypeZLib:
			zlibReader, err := zlib.NewReader(bytes.NewReader(compressedData))
			if err != nil {
				return fmt.Errorf("failed to create zlib reader: %w", err)
			}

			decompressedData, err = io.ReadAll(zlibReader)
			zlibReader.Close()
			if err != nil {
				return fmt.Errorf("failed to decompress with zlib: %w", err)
			}

		case CompTypePoly:
			decompressedData, err = decompressWithPolynomial(compressedData)
			if err != nil {
				return fmt.Errorf("failed to decompress with polynomial: %w", err)
			}

		case CompTypePolyZLib:
			zlibReader, err := zlib.NewReader(bytes.NewReader(compressedData))
			if err != nil {
				return fmt.Errorf("failed to create zlib reader: %w", err)
			}

			polyCompressed, err := io.ReadAll(zlibReader)
			zlibReader.Close()
			if err != nil {
				return fmt.Errorf("failed to decompress with zlib: %w", err)
			}

			decompressedData, err = decompressWithPolynomial(polyCompressed)
			if err != nil {
				return fmt.Errorf("failed to decompress with polynomial: %w", err)
			}

		default:
			return fmt.Errorf("unknown compression type: %d", compressionType)
		}

		// Write decompressed data
		_, err = bufWriter.Write(decompressedData)
		if err != nil {
			return fmt.Errorf("failed to write decompressed data: %w", err)
		}

		bytesWritten += int64(len(decompressedData))

		// Show progress every 10% or so
		if chunkID%(numChunks/10+1) == 0 {
			fmt.Printf("Progress: %.1f%% (chunk %d of %d)\n",
				float64(chunkID)/float64(numChunks)*100, chunkID+1, numChunks)
		}
	}

	// Flush any remaining data
	bufWriter.Flush()

	if bytesWritten != fileSize {
		fmt.Printf("Warning: Decompressed size (%d bytes) doesn't match expected size (%d bytes)\n",
			bytesWritten, fileSize)
	}

	fmt.Printf("Decompressed %s to %s (%d bytes)\n", inputPath, outputPath, bytesWritten)
	return nil
}

// decompressWithPolynomial decompresses data using polynomial method
func decompressWithPolynomial(data []byte) ([]byte, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("data too short for polynomial decompression")
	}

	// Read degree
	degree := int(data[0])
	if degree < 1 || degree > MaxDegree {
		return nil, fmt.Errorf("invalid polynomial degree: %d", degree)
	}

	// Read coefficients
	coeffs := make([]float64, degree+1)
	reader := bytes.NewReader(data[1:])

	for i := range coeffs {
		var scaled int16
		if err := binary.Read(reader, binary.LittleEndian, &scaled); err != nil {
			return nil, fmt.Errorf("failed to read coefficient: %w", err)
		}
		coeffs[i] = float64(scaled) / 100.0
	}

	// Read residuals
	residualsStart := 1 + (degree+1)*2 // 1 byte for degree + 2 bytes per coefficient
	if len(data) <= residualsStart {
		return nil, fmt.Errorf("data too short to contain residuals")
	}

	residuals := data[residualsStart:]

	// Reconstruct the original data
	result := make([]byte, len(residuals))
	for i := 0; i < len(residuals); i++ {
		// Normalized x
		x := float64(i) / float64(len(residuals)-1)

		// Evaluate polynomial
		predicted := evaluatePolynomial(coeffs, x)

		// Add residual
		error := int(residuals[i]) - 128 // Shift back from unsigned
		value := predicted + float64(error)

		// Clamp to byte range and round
		if value < 0 {
			value = 0
		} else if value > 255 {
			value = 255
		}

		result[i] = byte(math.Round(value))
	}

	return result, nil
}
