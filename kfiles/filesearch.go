package kfiles

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ledongthuc/pdf"
)

// FileDescription represents a file's semantic description
type FileDescription struct {
	FileID      string    `json:"file_id"`
	Description string    `json:"description"`
	Keywords    []string  `json:"keywords"`
	Summary     string    `json:"summary"`
	ContentType string    `json:"content_type"`
	GeneratedAt time.Time `json:"generated_at"`
}

// SearchResult represents a file search result with relevance score
type SearchResult struct {
	File      FileType `json:"file"`
	Score     float64  `json:"score"`
	MatchType string   `json:"match_type"` // "exact", "partial", "semantic"
	Excerpt   string   `json:"excerpt"`
}

// LLMRequest represents a request to the local LLM
type LLMRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// LLMResponse represents a response from the local LLM
type LLMResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// ContentExtractor handles different file types for text extraction
type ContentExtractor struct{}

// ExtractText extracts text content from various file types
func (ce *ContentExtractor) ExtractText(filePath string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".txt", ".md", ".json", ".yaml", ".yml", ".csv":
		return ce.extractPlainText(filePath)
	case ".pdf":
		return ce.extractPDFText(filePath)
	case ".docx":
		return ce.extractDocxText(filePath)
	default:
		// For other file types, try to read as plain text or return type-based description
		if content, err := ce.extractPlainText(filePath); err == nil && len(content) > 0 {
			return content, nil
		}
		return fmt.Sprintf("Binary file of type %s", ext), nil
	}
}

// extractPlainText reads text from plain text files
func (ce *ContentExtractor) extractPlainText(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	// Limit content size for processing (first 10KB)
	maxSize := 10240
	if len(content) > maxSize {
		content = content[:maxSize]
	}

	return string(content), nil
}

// extractPDFText extracts text from PDF files
func (ce *ContentExtractor) extractPDFText(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Get file info to obtain size
	fileInfo, err := f.Stat()
	if err != nil {
		return "", err
	}

	pdfReader, err := pdf.NewReader(f, fileInfo.Size())
	if err != nil {
		return "", err
	}

	var text strings.Builder
	maxPages := 5 // Limit to first 5 pages for performance

	for i := 1; i <= pdfReader.NumPage() && i <= maxPages; i++ {
		page := pdfReader.Page(i)
		if page.V.IsNull() {
			continue
		}

		pageText, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}

		text.WriteString(pageText)
		text.WriteString("\n")
	}

	return text.String(), nil
}

// extractDocxText extracts text from DOCX files (basic implementation)
func (ce *ContentExtractor) extractDocxText(filePath string) (string, error) {
	// This is a simplified implementation
	// In production, you'd use a proper DOCX parser like "github.com/nguyenthenguyen/docx"
	return fmt.Sprintf("Microsoft Word document: %s", filepath.Base(filePath)), nil
}

// LLMProcessor handles local LLM interactions
type LLMProcessor struct {
	OllamaURL   string
	OllamaModel string
	GeminiKey   string
	ModelType   string // "ollama" or "gemini"
}

// GeminiRequest represents a request to the Gemini API
type GeminiRequest struct {
	Contents []GeminiContent `json:"contents"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text string `json:"text"`
}

// GeminiResponse represents a response from the Gemini API
type GeminiResponse struct {
	Candidates []GeminiCandidate `json:"candidates"`
}

type GeminiCandidate struct {
	Content GeminiContent `json:"content"`
}
type AppSettings struct {
	GeminiAPIKey string `json:"gemini_api_key"`
	OllamaURL    string `json:"ollama_url"`
	OllamaModel  string `json:"ollama_model"`
}

// NewLLMProcessor creates a new LLM processor instance
func NewLLMProcessor(modelType string) *LLMProcessor {
	// Default settings
	processor := &LLMProcessor{
		OllamaURL:   "http://localhost:11434",
		OllamaModel: "phi3:mini",
		ModelType:   modelType,
	}

	// Try to load settings from JSON file
	homeDir, err := GetPlatformSpecificPath()
	if err == nil {
		jsonPath := filepath.Join(homeDir, "settings.json")

		// Read the JSON file
		if jsonData, err := os.ReadFile(jsonPath); err == nil {
			var settings AppSettings
			if err := json.Unmarshal(jsonData, &settings); err == nil {
				// Apply settings from JSON
				if settings.OllamaURL != "" {
					processor.OllamaURL = settings.OllamaURL
				}
				if settings.OllamaModel != "" {
					processor.OllamaModel = settings.OllamaModel
				}
				if modelType == "gemini" && settings.GeminiAPIKey != "" {
					processor.GeminiKey = settings.GeminiAPIKey
				}
			} else {
				log.Printf("Warning: Failed to parse settings.json: %v", err)
			}
		}
	}

	// Fallback to environment variables if not found in JSON
	if modelType == "gemini" && processor.GeminiKey == "" {
		processor.GeminiKey = os.Getenv("GEMINI_API_KEY")
		if processor.GeminiKey == "" {
			log.Printf("Warning: GEMINI_API_KEY not found in settings.json or environment variables")
		}
	}

	return processor
}

// GenerateDescription generates a description for file content using the selected LLM
func (llm *LLMProcessor) GenerateDescription(content string, fileName string) (FileDescription, error) {
	if llm.ModelType == "ollama" {
		// Use Ollama
		if !llm.isOllamaAvailable() {
			// Fallback to simple text analysis
			return llm.generateSimpleDescription(content, fileName), nil
		}

		prompt := llm.createPrompt(content, fileName)
		response, err := llm.callOllama(prompt)
		if err != nil {
			// Fallback to simple description if LLM fails
			return llm.generateSimpleDescription(content, fileName), nil
		}

		return llm.parseResponse(response, fileName), nil

	} else if llm.ModelType == "gemini" {
		// Use Gemini
		if llm.GeminiKey == "" {
			// Fallback to simple text analysis if no API key
			return llm.generateSimpleDescription(content, fileName), nil
		}

		prompt := llm.createPrompt(content, fileName)
		response, err := llm.callGemini(prompt)
		if err != nil {
			// Fallback to simple description if Gemini fails
			log.Printf("Gemini API call failed: %v", err)
			return llm.generateSimpleDescription(content, fileName), nil
		}

		return llm.parseResponse(response, fileName), nil

	} else {
		// Default to simple description for unknown model types
		return llm.generateSimpleDescription(content, fileName), nil
	}
}

// createPrompt creates a standardized prompt for both LLM providers
func (llm *LLMProcessor) createPrompt(content string, fileName string) string {
	return fmt.Sprintf(`Analyze the following file content and provide a brief description:

File: %s
Content: %s

Please provide:
1. A brief summary (max 100 words)
2. Key topics or themes
3. Important keywords (comma-separated)

Format your response as:
SUMMARY: [summary here]
TOPICS: [main topics]
KEYWORDS: [keyword1, keyword2, keyword3]`, fileName, llm.truncateContent(content))
}

// callGemini makes a request to the Gemini API
func (llm *LLMProcessor) callGemini(prompt string) (string, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=%s", llm.GeminiKey)

	reqBody := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{Text: prompt},
				},
			},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Gemini request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create Gemini request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call Gemini API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Gemini API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read Gemini response: %w", err)
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal Gemini response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from Gemini API")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

// isOllamaAvailable checks if Ollama is running
func (llm *LLMProcessor) isOllamaAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", llm.OllamaURL+"/api/tags", nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}

// callOllama makes a request to the local Ollama instance
func (llm *LLMProcessor) callOllama(prompt string) (string, error) {
	reqBody := LLMRequest{
		Model:  llm.OllamaModel,
		Prompt: prompt,
		Stream: false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", llm.OllamaURL+"/api/generate", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var llmResp LLMResponse
	if err := json.Unmarshal(body, &llmResp); err != nil {
		return "", err
	}

	return llmResp.Response, nil
}

// truncateContent limits content size for LLM processing
func (llm *LLMProcessor) truncateContent(content string) string {
	maxChars := 2000 // Limit for small models
	if len(content) <= maxChars {
		return content
	}
	return content[:maxChars] + "..."
}

// parseResponse parses LLM response into structured data
func (llm *LLMProcessor) parseResponse(response string, fileName string) FileDescription {
	lines := strings.Split(response, "\n")

	var summary, topics, keywordsStr string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "SUMMARY:") {
			summary = strings.TrimSpace(strings.TrimPrefix(line, "SUMMARY:"))
		} else if strings.HasPrefix(line, "TOPICS:") {
			topics = strings.TrimSpace(strings.TrimPrefix(line, "TOPICS:"))
		} else if strings.HasPrefix(line, "KEYWORDS:") {
			keywordsStr = strings.TrimSpace(strings.TrimPrefix(line, "KEYWORDS:"))
		}
	}

	// Parse keywords
	var keywords []string
	if keywordsStr != "" {
		for _, keyword := range strings.Split(keywordsStr, ",") {
			keywords = append(keywords, strings.TrimSpace(keyword))
		}
	}

	// Combine summary and topics for description
	description := summary
	if topics != "" {
		description += " Topics: " + topics
	}

	return FileDescription{
		Description: description,
		Keywords:    keywords,
		Summary:     summary,
		ContentType: "text",
		GeneratedAt: time.Now(),
	}
}

// generateSimpleDescription creates a basic description without LLM
func (llm *LLMProcessor) generateSimpleDescription(content string, fileName string) FileDescription {
	// Extract basic keywords using simple text analysis
	words := regexp.MustCompile(`\b\w+\b`).FindAllString(strings.ToLower(content), -1)

	// Count word frequency
	wordCount := make(map[string]int)
	for _, word := range words {
		if len(word) > 3 { // Only count words longer than 3 characters
			wordCount[word]++
		}
	}

	// Get top keywords
	var keywords []string
	for word, count := range wordCount {
		if count > 1 && len(keywords) < 10 {
			keywords = append(keywords, word)
		}
	}

	// Generate simple summary
	summary := fmt.Sprintf("File contains text with %d words. ", len(words))
	if len(keywords) > 0 {
		summary += "Key terms: " + strings.Join(keywords[:min(5, len(keywords))], ", ")
	}

	return FileDescription{
		Description: summary,
		Keywords:    keywords,
		Summary:     summary,
		ContentType: "text",
		GeneratedAt: time.Now(),
	}
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Add these methods to the FileBrowser struct

// GenerateFileDescription processes file content and generates a description
func (fb *FileBrowser) GenerateFileDescription(fileID string, filePath string, fileName string, modelType string) error {
	extractor := &ContentExtractor{}
	processor := NewLLMProcessor(modelType)

	// Extract text content
	content, err := extractor.ExtractText(filePath)
	if err != nil {
		return fmt.Errorf("failed to extract content: %w", err)
	}

	// Generate description using selected LLM
	description, err := processor.GenerateDescription(content, fileName)
	if err != nil {
		return fmt.Errorf("failed to generate description: %w", err)
	}

	description.FileID = fileID

	// Store description in database
	return fb.storeFileDescription(description)
}

// GenerateFileDescriptionWithModel allows external specification of model type
func (fb *FileBrowser) GenerateFileDescriptionWithModel(fileID string, filePath string, fileName string, modelType string) error {
	return fb.GenerateFileDescription(fileID, filePath, fileName, modelType)
}

// storeFileDescription saves the generated description to database
func (fb *FileBrowser) storeFileDescription(desc FileDescription) error {
	db, err := sql.Open("sqlite3", fb.DbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Create table if not exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS file_descriptions (
			file_id TEXT PRIMARY KEY,
			description TEXT NOT NULL,
			summary TEXT,
			keywords TEXT,
			content_type TEXT,
			generated_at TIMESTAMP NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create descriptions table: %w", err)
	}

	// Convert keywords to JSON
	keywordsJSON, _ := json.Marshal(desc.Keywords)

	// Insert or update description
	_, err = db.Exec(`
		INSERT OR REPLACE INTO file_descriptions
		(file_id, description, summary, keywords, content_type, generated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, desc.FileID, desc.Description, desc.Summary, string(keywordsJSON), desc.ContentType, desc.GeneratedAt)

	if err != nil {
		return fmt.Errorf("failed to store description: %w", err)
	}

	return nil
}

// SearchFiles searches for files based on a natural language query
func (fb *FileBrowser) SearchFiles(query string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 20
	}

	db, err := sql.Open("sqlite3", fb.DbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	var results []SearchResult

	// Search in file descriptions and keywords
	searchQuery := `
		SELECT
			f.file_id, f.file_name, f.file_path, f.file_size, f.created_at, f.updated_at, f.is_distributed,
			fd.description, fd.summary, fd.keywords
		FROM files f
		LEFT JOIN file_descriptions fd ON f.file_id = fd.file_id
		WHERE
			fd.description LIKE ? OR
			fd.summary LIKE ? OR
			fd.keywords LIKE ? OR
			f.file_name LIKE ?
		ORDER BY
			CASE
				WHEN f.file_name LIKE ? THEN 1
				WHEN fd.description LIKE ? THEN 2
				WHEN fd.keywords LIKE ? THEN 3
				ELSE 4
			END,
			f.updated_at DESC
		LIMIT ?
	`

	likeQuery := "%" + query + "%"

	rows, err := db.Query(searchQuery,
		likeQuery, likeQuery, likeQuery, likeQuery, // WHERE clauses
		likeQuery, likeQuery, likeQuery, // ORDER BY clauses
		limit)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var fileID, fileName, filePath, description, summary, keywordsJSON string
		var fileSize int64
		var createdAt, updatedAt time.Time
		var isDistributed bool

		err := rows.Scan(&fileID, &fileName, &filePath, &fileSize, &createdAt, &updatedAt, &isDistributed,
			&description, &summary, &keywordsJSON)
		if err != nil {
			continue
		}

		// Calculate relevance score
		score := fb.calculateRelevanceScore(query, fileName, description, summary, keywordsJSON)

		// Determine match type
		matchType := "partial"
		if strings.Contains(strings.ToLower(fileName), strings.ToLower(query)) {
			matchType = "exact"
		} else if score > 0.8 {
			matchType = "semantic"
		}

		// Generate excerpt
		excerpt := fb.generateExcerpt(query, description, summary)

		file := FileType{
			ID:               fileID,
			Name:             fileName,
			Type:             "file",
			Path:             filePath,
			SizeInBytes:      fileSize,
			Size:             formatSize(fileSize),
			LastModified:     formatLastModified(updatedAt),
			LastModifiedDate: updatedAt,
			Owner:            "You",
			IsShared:         false,
			IsDistributed:    isDistributed,
		}

		result := SearchResult{
			File:      file,
			Score:     score,
			MatchType: matchType,
			Excerpt:   excerpt,
		}

		results = append(results, result)
	}

	return results, nil
}

// calculateRelevanceScore calculates how relevant a file is to the query
func (fb *FileBrowser) calculateRelevanceScore(query, fileName, description, summary, keywordsJSON string) float64 {
	query = strings.ToLower(query)
	fileName = strings.ToLower(fileName)
	description = strings.ToLower(description)
	summary = strings.ToLower(summary)

	score := 0.0

	// Exact filename match gets highest score
	if strings.Contains(fileName, query) {
		score += 1.0
	}

	// Description match
	if strings.Contains(description, query) {
		score += 0.8
	}

	// Summary match
	if strings.Contains(summary, query) {
		score += 0.6
	}

	// Keywords match
	var keywords []string
	json.Unmarshal([]byte(keywordsJSON), &keywords)
	for _, keyword := range keywords {
		if strings.Contains(strings.ToLower(keyword), query) {
			score += 0.4
		}
	}

	// Word-level matching for partial matches
	queryWords := strings.Fields(query)
	for _, word := range queryWords {
		if strings.Contains(description, word) {
			score += 0.2
		}
		if strings.Contains(fileName, word) {
			score += 0.3
		}
	}

	// Normalize score to 0-1 range
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// generateExcerpt creates a text excerpt highlighting the match
func (fb *FileBrowser) generateExcerpt(query, description, summary string) string {
	text := description
	if text == "" {
		text = summary
	}

	if text == "" {
		return "No description available"
	}

	query = strings.ToLower(query)
	textLower := strings.ToLower(text)

	// Find the position of the query in the text
	pos := strings.Index(textLower, query)
	if pos == -1 {
		// If exact query not found, look for individual words
		words := strings.Fields(query)
		for _, word := range words {
			pos = strings.Index(textLower, word)
			if pos != -1 {
				break
			}
		}
	}

	if pos == -1 {
		// If no match found, return beginning of text
		if len(text) > 150 {
			return text[:150] + "..."
		}
		return text
	}

	// Extract excerpt around the match
	start := pos - 50
	if start < 0 {
		start = 0
	}

	end := pos + len(query) + 50
	if end > len(text) {
		end = len(text)
	}

	excerpt := text[start:end]
	if start > 0 {
		excerpt = "..." + excerpt
	}
	if end < len(text) {
		excerpt = excerpt + "..."
	}

	return excerpt
}
