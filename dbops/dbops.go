package dbops


package dbops

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB is the global database connection
var DB *sql.DB

// DBConfig holds database configuration
type DBConfig struct {
	Path string // Path to SQLite database file
}

// Initialize initializes the database connection
func Initialize(config DBConfig) error {
	var err error
	DB, err = sql.Open("sqlite3", config.Path)
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}

	// Test the connection
	if err = DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}

	// Create tables if they don't exist
	if err = createTables(); err != nil {
		return fmt.Errorf("failed to create tables: %v", err)
	}

	return nil
}

// Close closes the database connection
func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

// createTables creates the necessary tables if they don't exist
func createTables() error {
	// Create users table
	_, err := DB.Exec(`
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		email TEXT UNIQUE NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		modified_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return err
	}

	// Create face_features table
	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS face_features (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		features BLOB NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(user_id) REFERENCES users(id)
	)`)
	if err != nil {
		return err
	}

	return nil
}

// StoreData inserts or updates data in the specified table
func StoreData(tableName string, data map[string]interface{}) (string, error) {
	if DB == nil {
		return "", errors.New("database not initialized")
	}

	// Extract keys and values from the data map
	var keys []string
	var placeholders []string
	var values []interface{}
	var updateParts []string

	id, hasID := data["id"].(string)
	if !hasID || id == "" {
		return "", errors.New("id field is required and must be a string")
	}

	for k, v := range data {
		keys = append(keys, k)
		placeholders = append(placeholders, "?")
		values = append(values, v)

		if k != "id" { // Don't update the ID field
			updateParts = append(updateParts, fmt.Sprintf("%s = ?", k))
		}
	}

	// Add modified_at if table has that column (for updates)
	if tableName == "users" {
		keys = append(keys, "modified_at")
		placeholders = append(placeholders, "?")
		values = append(values, time.Now())
		updateParts = append(updateParts, "modified_at = ?")
	}

	// Construct the query
	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT(id) DO UPDATE SET %s",
		tableName,
		strings.Join(keys, ", "),
		strings.Join(placeholders, ", "),
		strings.Join(updateParts, ", "),
	)

	// For update clause, we need to append values again (except id)
	for i, k := range keys {
		if k != "id" {
			values = append(values, values[i])
		}
	}

	// Execute the query
	_, err := DB.Exec(query, values...)
	if err != nil {
		return "", fmt.Errorf("error storing data: %v", err)
	}

	return id, nil
}

// GetData retrieves data from the specified table
func GetData(tableName, id string) (map[string]interface{}, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}

	// Get table columns
	rows, err := DB.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return nil, fmt.Errorf("error getting table info: %v", err)
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var cid, notnull, pk int
		var name, type_name string
		var dflt_value interface{}
		if err := rows.Scan(&cid, &name, &type_name, &notnull, &dflt_value, &pk); err != nil {
			return nil, err
		}
		columns = append(columns, name)
	}

	// Construct the query
	query := fmt.Sprintf("SELECT %s FROM %s WHERE id = ?", strings.Join(columns, ", "), tableName)
	row := DB.QueryRow(query, id)

	// Create a slice to hold the values
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	// Scan the row into the values
	if err := row.Scan(valuePtrs...); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No record found
		}
		return nil, fmt.Errorf("error scanning row: %v", err)
	}

	// Create a map from the values
	result := make(map[string]interface{})
	for i, col := range columns {
		val := values[i]
		result[col] = val
	}

	return result, nil
}

// GetAllData retrieves all data from the specified table that matches the filter
func GetAllData(tableName string, filter map[string]interface{}) ([]map[string]interface{}, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}

	// Get table columns
	rows, err := DB.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return nil, fmt.Errorf("error getting table info: %v", err)
	}

	var columns []string
	for rows.Next() {
		var cid, notnull, pk int
		var name, type_name string
		var dflt_value interface{}
		if err := rows.Scan(&cid, &name, &type_name, &notnull, &dflt_value, &pk); err != nil {
			return nil, err
		}
		columns = append(columns, name)
	}
	rows.Close()

	// Construct the query with filters
	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(columns, ", "), tableName)
	var conditions []string
	var values []interface{}

	if len(filter) > 0 {
		query += " WHERE "
		for k, v := range filter {
			conditions = append(conditions, fmt.Sprintf("%s = ?", k))
			values = append(values, v)
		}
		query += strings.Join(conditions, " AND ")
	}

	// Execute the query
	rows, err = DB.Query(query, values...)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %v", err)
	}
	defer rows.Close()

	// Process results
	var results []map[string]interface{}
	for rows.Next() {
		// Create a slice to hold the values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// Scan the row into the values
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("error scanning row: %v", err)
		}

		// Create a map from the values
		result := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			result[col] = val
		}

		results = append(results, result)
	}

	return results, nil
}

// DeleteData deletes data from the specified table
func DeleteData(tableName, id string) error {
	if DB == nil {
		return errors.New("database not initialized")
	}

	// Execute the delete query
	_, err := DB.Exec(fmt.Sprintf("DELETE FROM %s WHERE id = ?", tableName), id)
	if err != nil {
		return fmt.Errorf("error deleting data: %v", err)
	}

	return nil
}

// Specific functions for facial recognition

// StoreFaceFeatures stores facial features for a user
func StoreFaceFeatures(userID string, features []float64) (string, error) {
	if DB == nil {
		return "", errors.New("database not initialized")
	}

	// Convert features to JSON for storage
	featuresBytes, err := json.Marshal(features)
	if err != nil {
		return "", fmt.Errorf("error marshaling features: %v", err)
	}

	// Generate a feature ID
	featureID := fmt.Sprintf("feat_%d", time.Now().UnixNano())

	// Store in face_features table
	_, err = DB.Exec(
		"INSERT INTO face_features (id, user_id, features) VALUES (?, ?, ?)",
		featureID, userID, featuresBytes,
	)
	if err != nil {
		return "", fmt.Errorf("error storing face features: %v", err)
	}

	return featureID, nil
}

// GetUserFaceFeatures retrieves facial features for a specific user
func GetUserFaceFeatures(userID string) ([]float64, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}

	// Query the latest face features for the user
	var featuresBytes []byte
	err := DB.QueryRow(
		"SELECT features FROM face_features WHERE user_id = ? ORDER BY created_at DESC LIMIT 1",
		userID,
	).Scan(&featuresBytes)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No features found
		}
		return nil, fmt.Errorf("error retrieving face features: %v", err)
	}

	// Unmarshal features from JSON
	var features []float64
	if err := json.Unmarshal(featuresBytes, &features); err != nil {
		return nil, fmt.Errorf("error unmarshaling features: %v", err)
	}

	return features, nil
}

// GetAllUsersFaceFeatures retrieves facial features for all users
func GetAllUsersFaceFeatures() (map[string][]float64, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}

	// Query to get the latest face features for each user
	rows, err := DB.Query(`
		SELECT u.id, u.email, ff.features
		FROM users u
		JOIN face_features ff ON u.id = ff.user_id
		WHERE ff.id IN (
			SELECT MAX(id)
			FROM face_features
			GROUP BY user_id
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("error retrieving users face features: %v", err)
	}
	defer rows.Close()

	// Process results
	userFeatures := make(map[string][]float64)
	for rows.Next() {
		var userID, email string
		var featuresBytes []byte
		if err := rows.Scan(&userID, &email, &featuresBytes); err != nil {
			return nil, fmt.Errorf("error scanning row: %v", err)
		}

		var features []float64
		if err := json.Unmarshal(featuresBytes, &features); err != nil {
			return nil, fmt.Errorf("error unmarshaling features for user %s: %v", email, err)
		}

		userFeatures[userID] = features
	}

	return userFeatures, nil
}

// GetUserByEmail retrieves a user by email
func GetUserByEmail(email string) (map[string]interface{}, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}

	// Get user by email
	row := DB.QueryRow("SELECT id, email, created_at, modified_at FROM users WHERE email = ?", email)

	var user map[string]interface{} = make(map[string]interface{})
	var id, userEmail string
	var createdAt, modifiedAt time.Time

	if err := row.Scan(&id, &userEmail, &createdAt, &modifiedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No user found
		}
		return nil, fmt.Errorf("error scanning user row: %v", err)
	}

	user["id"] = id
	user["email"] = userEmail
	user["created_at"] = createdAt
	user["modified_at"] = modifiedAt

	return user, nil
}

// CreateUser creates a new user
func CreateUser(email string) (string, error) {
	if DB == nil {
		return "", errors.New("database not initialized")
	}

	// Check if user already exists
	existing, err := GetUserByEmail(email)
	if err != nil {
		return "", err
	}
	if existing != nil {
		return existing["id"].(string), nil // User already exists
	}

	// Generate user ID
	userID := fmt.Sprintf("user_%d", time.Now().UnixNano())

	// Create user
	_, err = DB.Exec(
		"INSERT INTO users (id, email) VALUES (?, ?)",
		userID, email,
	)
	if err != nil {
		return "", fmt.Errorf("error creating user: %v", err)
	}

	return userID, nil
}

// GetUserByFaceFeatures finds a user by comparing face features
func GetUserByFaceFeatures(features []float64, similarityThreshold float64) (string, error) {
	if DB == nil {
		return "", errors.New("database not initialized")
	}

	// Get all users' face features
	allUserFeatures, err := GetAllUsersFaceFeatures()
	if err != nil {
		return "", err
	}

	// Find the best match
	var bestMatchUserID string
	var bestSimilarity float64 = -1 // Cosine similarity ranges from -1 to 1

	for userID, userFeatures := range allUserFeatures {
		similarity := calculateCosineSimilarity(features, userFeatures)
		if similarity > bestSimilarity {
			bestSimilarity = similarity
			bestMatchUserID = userID
		}
	}

	// Check if similarity exceeds threshold
	if bestSimilarity >= similarityThreshold {
		// Get user email
		userData, err := GetData("users", bestMatchUserID)
		if err != nil {
			return "", err
		}
		if userData == nil {
			return "", fmt.Errorf("user not found for ID: %s", bestMatchUserID)
		}
		return userData["email"].(string), nil
	}

	return "", fmt.Errorf("no matching user found (best match: %.2f, threshold: %.2f)", bestSimilarity, similarityThreshold)
}

// calculateCosineSimilarity calculates the cosine similarity between two feature vectors
func calculateCosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return -1 // Error case
	}

	var dotProduct, magnitudeA, magnitudeB float64

	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		magnitudeA += a[i] * a[i]
		magnitudeB += b[i] * b[i]
	}

	magnitudeA = float64(magnitudeA) * float64(magnitudeB)

	if magnitudeA == 0 {
		return -1
	}

	return dotProduct / magnitudeA
}
