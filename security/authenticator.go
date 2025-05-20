package security

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

const (
	DefaultAdminEmail    = "admin@kube.local"
	DefaultAdminPassword = "KubeAdmin123!" // Change this in production
)

// User represents a user in the system
type User struct {
	ID            int        `json:"id"`
	Email         string     `json:"email"`
	PasswordHash  string     `json:"-"` // Don't include in JSON responses
	FacialDataID  string     `json:"facial_data_id,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	LastLoginAt   *time.Time `json:"last_login_at,omitempty"`
	IsActive      bool       `json:"is_active"`
	HasFacialAuth bool       `json:"has_facial_auth"`
	IsAdmin       bool       `json:"is_admin"`
}

// UserManager handles user authentication and management
type UserManager struct {
	dbPath       string
	facialSystem *FacialSystem
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RegistrationRequest represents a user registration request
type RegistrationRequest struct {
	Email            string   `json:"email"`
	Password         string   `json:"password"`
	EnableFacialAuth bool     `json:"enable_facial_auth"`
	FaceFrames       []string `json:"face_frames,omitempty"`
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	Success   bool   `json:"success"`
	User      *User  `json:"user,omitempty"`
	Token     string `json:"token,omitempty"`
	Message   string `json:"message,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

// NewUserManager creates a new UserManager instance
func NewUserManager(dbPath string, facialSystem *FacialSystem) (*UserManager, error) {
	um := &UserManager{
		dbPath:       dbPath,
		facialSystem: facialSystem,
	}

	// Initialize the database
	if err := um.initDatabase(); err != nil {
		return nil, fmt.Errorf("failed to initialize user database: %w", err)
	}

	return um, nil
}

// ADMIN ENTRIES
func (um *UserManager) createDefaultAdmin() error {
	db, err := sql.Open("sqlite3", um.dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Check if any admin user already exists
	var adminCount int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE is_admin = 1").Scan(&adminCount)
	if err != nil {
		return fmt.Errorf("failed to check for existing admin: %w", err)
	}

	// If admin already exists, skip creation
	if adminCount > 0 {
		log.Println("Admin user already exists, skipping default admin creation")
		return nil
	}

	// Check if default admin email is already taken by a non-admin user
	var existingUserID int
	err = db.QueryRow("SELECT id FROM users WHERE email = ?", DefaultAdminEmail).Scan(&existingUserID)
	if err == nil {
		// User exists, promote to admin
		_, err = db.Exec("UPDATE users SET is_admin = 1 WHERE id = ?", existingUserID)
		if err != nil {
			return fmt.Errorf("failed to promote existing user to admin: %w", err)
		}
		log.Printf("Promoted existing user %s to admin", DefaultAdminEmail)
		return nil
	} else if err != sql.ErrNoRows {
		return fmt.Errorf("failed to check for existing user: %w", err)
	}

	// Hash the default admin password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(DefaultAdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash admin password: %w", err)
	}

	// Create the admin user
	_, err = db.Exec(`
		INSERT INTO users (email, password_hash, is_active, has_facial_auth, is_admin)
		VALUES (?, ?, 1, 0, 1)
	`, DefaultAdminEmail, string(hashedPassword))
	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	log.Printf("Created default admin user: %s", DefaultAdminEmail)
	log.Printf("Default admin password: %s", DefaultAdminPassword)
	log.Println("IMPORTANT: Please change the admin password on first login!")

	return nil
}

// IsUserAdmin checks if a user has admin privileges
func (um *UserManager) IsUserAdmin(userID int) (bool, error) {
	db, err := sql.Open("sqlite3", um.dbPath)
	if err != nil {
		return false, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	var isAdmin bool
	err = db.QueryRow("SELECT is_admin FROM users WHERE id = ? AND is_active = 1", userID).Scan(&isAdmin)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to check admin status: %w", err)
	}

	return isAdmin, nil
}

// initDatabase creates the necessary tables
func (um *UserManager) initDatabase() error {
	// Ensure directory exists
	dir := filepath.Dir(um.dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite3", um.dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Create users table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT,
			facial_data_id TEXT UNIQUE,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_login_at TIMESTAMP,
			is_active BOOLEAN NOT NULL DEFAULT 1,
			has_facial_auth BOOLEAN NOT NULL DEFAULT 0,
			is_admin BOOLEAN NOT NULL DEFAULT 0
		);

		CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
		CREATE INDEX IF NOT EXISTS idx_users_admin ON users(is_admin);
		CREATE INDEX IF NOT EXISTS idx_users_facial_data_id ON users(facial_data_id);
	`)
	if err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS face_features (
			id TEXT PRIMARY KEY,
			features BLOB NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (id) REFERENCES users(facial_data_id) ON DELETE CASCADE
		)`)
	if err != nil {
		return fmt.Errorf("failed to create facefeature table: %w", err)
	}

	_, err = db.Exec(`
    CREATE TABLE IF NOT EXISTS compress_logs (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        in_file_path TEXT NOT NULL,
        compressed_file_path TEXT NOT NULL
    );`)
	if err != nil {
		return fmt.Errorf("failed to create compression logs table: %w", err)
	}

	// Create sessions table for session management
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS user_sessions (
			session_id TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP NOT NULL,
			is_active BOOLEAN NOT NULL DEFAULT 1,
			FOREIGN KEY (user_id) REFERENCES users (id)
		);

		CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON user_sessions(user_id);
		CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON user_sessions(expires_at);
	`)
	if err != nil {
		return fmt.Errorf("failed to create sessions table: %w", err)
	}

	if err := um.createDefaultAdmin(); err != nil {
		return fmt.Errorf("failed to create default admin: %w", err)
	}

	return nil
}

// RegisterUser registers a new user with optional facial authentication
func (um *UserManager) RegisterUser(req RegistrationRequest) (*AuthResponse, error) {
	// Validate input
	if req.Email == "" {
		return &AuthResponse{
			Success: false,
			Message: "Email is required",
		}, nil
	}

	if req.Password == "" && !req.EnableFacialAuth {
		return &AuthResponse{
			Success: false,
			Message: "Password is required when facial authentication is not enabled",
		}, nil
	}

	// Check if user already exists
	existingUser, err := um.getUserByEmail(req.Email)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return &AuthResponse{
			Success: false,
			Message: "User with this email already exists",
		}, nil
	}

	// Hash password if provided
	var passwordHash string
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}
		passwordHash = string(hash)
	}

	// Handle facial authentication if enabled
	// var facialDataID string
	// if req.EnableFacialAuth && um.facialSystem != nil {
	// 	println("training the user")
	// 	faceID, err := um.facialSystem.TrainNewUser(req.Email)
	// 	if err != nil {
	// 		return &AuthResponse{
	// 			Success: false,
	// 			Message: fmt.Sprintf("Failed to register facial data: %v", err),
	// 		}, nil
	// 	}
	// 	facialDataID = faceID
	// }
	var facialDataID string
	if req.EnableFacialAuth && um.facialSystem != nil {

		println("CHECKING FACIAL SYSTEM")
		faceID, err := um.facialSystem.TrainNewUserWithFrames(um.dbPath, req.FaceFrames)
		if err != nil {
			return &AuthResponse{
				Success: false,
				Message: fmt.Sprintf("Failed to register facial data: %v", err),
			}, nil
		}
		facialDataID = faceID

		// if len(req.FaceFrames) > 0 {
		// 	// Use provided frames from frontend
		// 	println("CHECKING FACIAL SYSTEM")
		// 	faceID, err := um.facialSystem.TrainNewUserWithFrames(req.FaceFrames)
		// 	if err != nil {
		// 		return &AuthResponse{
		// 			Success: false,
		// 			Message: fmt.Sprintf("Failed to register facial data: %v", err),
		// 		}, nil
		// 	}
		// 	facialDataID = faceID
		// } else {
		// 	// Fallback to camera-based training (existing method)
		// 	faceID, err := um.facialSystem.TrainNewUser(req.Email)
		// 	if err != nil {
		// 		return &AuthResponse{
		// 			Success: false,
		// 			Message: fmt.Sprintf("Failed to register facial data: %v", err),
		// 		}, nil
		// 	}
		// 	facialDataID = faceID
		// }
	}

	// Create user in database
	db, err := sql.Open("sqlite3", um.dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	result, err := db.Exec(`
		INSERT INTO users (email, password_hash, facial_data_id, has_facial_auth)
		VALUES (?, ?, ?, ?)
	`, req.Email, passwordHash, facialDataID, req.EnableFacialAuth)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	userID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID: %w", err)
	}

	// Get the created user
	user, err := um.getUserByID(int(userID))
	if err != nil {
		return nil, fmt.Errorf("failed to get created user: %w", err)
	}

	// Create session
	sessionID, err := um.createSession(user.ID)
	if err != nil {
		log.Printf("Warning: Failed to create session for new user: %v", err)
	}

	return &AuthResponse{
		Success:   true,
		User:      user,
		Message:   "User registered successfully",
		SessionID: sessionID,
	}, nil
}

// LoginWithCredentials authenticates user with email and password
func (um *UserManager) LoginWithCredentials(email, password string) (*AuthResponse, error) {
	if email == "" || password == "" {
		return &AuthResponse{
			Success: false,
			Message: "Email and password are required",
		}, nil
	}

	// Get user by email
	user, err := um.getUserByEmail(email)
	if err != nil {
		if err == sql.ErrNoRows {
			return &AuthResponse{
				Success: false,
				Message: "Invalid email or password",
			}, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check if user is active
	if !user.IsActive {
		return &AuthResponse{
			Success: false,
			Message: "Account is disabled",
		}, nil
	}

	// Verify password
	if user.PasswordHash == "" {
		return &AuthResponse{
			Success: false,
			Message: "Password authentication not available for this account",
		}, nil
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return &AuthResponse{
			Success: false,
			Message: "Invalid email or password",
		}, nil
	}

	// Update last login
	if err := um.updateLastLogin(user.ID); err != nil {
		log.Printf("Warning: Failed to update last login for user %d: %v", user.ID, err)
	}

	// Create session
	sessionID, err := um.createSession(user.ID)
	if err != nil {
		log.Printf("Warning: Failed to create session for user %d: %v", user.ID, err)
	}

	return &AuthResponse{
		Success:   true,
		User:      user,
		Message:   "Login successful",
		SessionID: sessionID,
	}, nil
}

// LoginWithFacialAuth authenticates user using facial recognition
func (um *UserManager) LoginWithFacialAuth(frame string) (*AuthResponse, error) {
	if um.facialSystem == nil {
		return &AuthResponse{
			Success: false,
			Message: "Facial authentication not available",
		}, nil
	}
	if frame == "" {
		return &AuthResponse{
			Success: false,
			Message: "No frame data provided",
		}, nil
	}

	// Perform facial recognition
	//frame := "frame" //ADJUST this
	email, err := um.facialSystem.LoginUserWithFrame(um.dbPath, frame)
	if err != nil {
		return &AuthResponse{
			Success: false,
			Message: fmt.Sprintf("Facial authentication failed: %v", err),
		}, nil
	}

	// Get user by email
	user, err := um.getUserByEmail(email)
	if err != nil {
		if err == sql.ErrNoRows {
			return &AuthResponse{
				Success: false,
				Message: "User not found",
			}, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check if user is active
	if !user.IsActive {
		return &AuthResponse{
			Success: false,
			Message: "Account is disabled",
		}, nil
	}

	// Check if user has facial authentication enabled
	if !user.HasFacialAuth {
		return &AuthResponse{
			Success: false,
			Message: "Facial authentication not enabled for this account",
		}, nil
	}

	// Update last login
	if err := um.updateLastLogin(user.ID); err != nil {
		log.Printf("Warning: Failed to update last login for user %d: %v", user.ID, err)
	}

	// Create session
	sessionID, err := um.createSession(user.ID)
	if err != nil {
		log.Printf("Warning: Failed to create session for user %d: %v", user.ID, err)
	}

	return &AuthResponse{
		Success:   true,
		User:      user,
		Message:   "Facial login successful",
		SessionID: sessionID,
	}, nil
}

// ValidateSession validates a session and returns the associated user
func (um *UserManager) ValidateSession(sessionID string) (*User, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID is required")
	}

	db, err := sql.Open("sqlite3", um.dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Get session and check if it's valid
	var userID int
	var expiresAt time.Time
	err = db.QueryRow(`
		SELECT user_id, expires_at
		FROM user_sessions
		WHERE session_id = ? AND is_active = 1
	`, sessionID).Scan(&userID, &expiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid session")
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Check if session is expired
	if time.Now().After(expiresAt) {
		// Deactivate expired session
		_, err = db.Exec(`
			UPDATE user_sessions
			SET is_active = 0
			WHERE session_id = ?
		`, sessionID)
		if err != nil {
			log.Printf("Warning: Failed to deactivate expired session: %v", err)
		}
		return nil, fmt.Errorf("session expired")
	}

	// Get user
	user, err := um.getUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// Logout invalidates a session
func (um *UserManager) Logout(sessionID string) error {
	if sessionID == "" {
		return nil // No session to logout
	}

	db, err := sql.Open("sqlite3", um.dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	_, err = db.Exec(`
		UPDATE user_sessions
		SET is_active = 0
		WHERE session_id = ?
	`, sessionID)
	if err != nil {
		return fmt.Errorf("failed to logout: %w", err)
	}

	return nil
}

// GetCurrentUser returns the current user for a session
func (um *UserManager) GetCurrentUser(sessionID string) (*User, error) {
	return um.ValidateSession(sessionID)
}

// getUserByEmail retrieves a user by email
func (um *UserManager) getUserByEmail_old(email string) (*User, error) {
	db, err := sql.Open("sqlite3", um.dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	user := &User{}
	var lastLoginAt sql.NullTime
	err = db.QueryRow(`
		SELECT id, email, password_hash, facial_data_id, created_at, last_login_at, is_active, has_facial_auth
		FROM users WHERE email = ?
	`, email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.FacialDataID,
		&user.CreatedAt, &lastLoginAt, &user.IsActive, &user.HasFacialAuth)
	if err != nil {
		return nil, err
	}

	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}

	return user, nil
}

// getUserByID retrieves a user by ID
func (um *UserManager) getUserByID_old(id int) (*User, error) {
	db, err := sql.Open("sqlite3", um.dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	user := &User{}
	var lastLoginAt sql.NullTime
	err = db.QueryRow(`
		SELECT id, email, password_hash, facial_data_id, created_at, last_login_at, is_active, has_facial_auth
		FROM users WHERE id = ?
	`, id).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.FacialDataID,
		&user.CreatedAt, &lastLoginAt, &user.IsActive, &user.HasFacialAuth)
	if err != nil {
		return nil, err
	}

	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}

	return user, nil
}

// getUserByEmail retrieves a user by email
func (um *UserManager) getUserByEmail(email string) (*User, error) {
	db, err := sql.Open("sqlite3", um.dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	user := &User{}
	var lastLoginAt sql.NullTime
	var facialDataID sql.NullString
	var passwordHash sql.NullString

	err = db.QueryRow(`
		SELECT id, email, password_hash, facial_data_id, created_at, last_login_at, is_active, has_facial_auth
		FROM users WHERE email = ?
	`, email).Scan(&user.ID, &user.Email, &passwordHash, &facialDataID,
		&user.CreatedAt, &lastLoginAt, &user.IsActive, &user.HasFacialAuth)
	if err != nil {
		return nil, err
	}

	// Handle NULL values
	if passwordHash.Valid {
		user.PasswordHash = passwordHash.String
	}
	if facialDataID.Valid {
		user.FacialDataID = facialDataID.String
	}
	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}

	return user, nil
}

// getUserByID retrieves a user by ID
func (um *UserManager) getUserByID(id int) (*User, error) {
	db, err := sql.Open("sqlite3", um.dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	user := &User{}
	var lastLoginAt sql.NullTime
	var facialDataID sql.NullString
	var passwordHash sql.NullString

	err = db.QueryRow(`
		SELECT id, email, password_hash, facial_data_id, created_at, last_login_at, is_active, has_facial_auth
		FROM users WHERE id = ?
	`, id).Scan(&user.ID, &user.Email, &passwordHash, &facialDataID,
		&user.CreatedAt, &lastLoginAt, &user.IsActive, &user.HasFacialAuth)
	if err != nil {
		return nil, err
	}

	// Handle NULL values
	if passwordHash.Valid {
		user.PasswordHash = passwordHash.String
	}
	if facialDataID.Valid {
		user.FacialDataID = facialDataID.String
	}
	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}

	return user, nil
}

// updateLastLogin updates the last login timestamp
func (um *UserManager) updateLastLogin(userID int) error {
	db, err := sql.Open("sqlite3", um.dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	_, err = db.Exec(`
		UPDATE users
		SET last_login_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, userID)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	return nil
}

// createSession creates a new session for a user
func (um *UserManager) createSession(userID int) (string, error) {
	db, err := sql.Open("sqlite3", um.dbPath)
	if err != nil {
		return "", fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Generate session ID (you might want to use a more secure method)
	sessionID := fmt.Sprintf("session_%d_%d", userID, time.Now().Unix())

	// Sessions expire in 30 days
	expiresAt := time.Now().Add(30 * 24 * time.Hour)

	_, err = db.Exec(`
		INSERT INTO user_sessions (session_id, user_id, expires_at)
		VALUES (?, ?, ?)
	`, sessionID, userID, expiresAt)
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	return sessionID, nil
}

// CleanupExpiredSessions removes expired sessions from the database
func (um *UserManager) CleanupExpiredSessions() error {
	db, err := sql.Open("sqlite3", um.dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	_, err = db.Exec(`
		DELETE FROM user_sessions
		WHERE expires_at < CURRENT_TIMESTAMP OR is_active = 0
	`)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}

	return nil
}

// ListUsers returns all registered users
func (um *UserManager) ListUsers() ([]*User, error) {
	db, err := sql.Open("sqlite3", um.dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	rows, err := db.Query(`
        SELECT id, email, facial_data_id, created_at, last_login_at, is_active, has_facial_auth, is_admin
        FROM users
        ORDER BY created_at DESC
    `)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	users := []*User{}
	for rows.Next() {
		user := &User{}
		var lastLoginAt sql.NullTime
		var facialDataID sql.NullString

		err := rows.Scan(
			&user.ID,
			&user.Email,
			&facialDataID,
			&user.CreatedAt,
			&lastLoginAt,
			&user.IsActive,
			&user.HasFacialAuth,
			&user.IsAdmin,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning user row: %w", err)
		}

		// Handle NULL values
		if facialDataID.Valid {
			user.FacialDataID = facialDataID.String
		}
		if lastLoginAt.Valid {
			user.LastLoginAt = &lastLoginAt.Time
		}

		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user rows: %w", err)
	}

	return users, nil
}

// DeleteUser removes a user and their facial data from the system
func (um *UserManager) DeleteUser(userID int) error {
	db, err := sql.Open("sqlite3", um.dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Start a transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Get user's facial data ID before deletion
	var facialDataID sql.NullString
	err = tx.QueryRow("SELECT facial_data_id FROM users WHERE id = ?", userID).Scan(&facialDataID)
	if err != nil {
		tx.Rollback()
		if err == sql.ErrNoRows {
			return fmt.Errorf("user not found")
		}
		return fmt.Errorf("failed to get user facial data: %w", err)
	}

	// Delete user sessions
	_, err = tx.Exec("DELETE FROM user_sessions WHERE user_id = ?", userID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}

	// If facial data exists, delete it
	if facialDataID.Valid && facialDataID.String != "" {
		_, err = tx.Exec("DELETE FROM face_features WHERE id = ?", facialDataID.String)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to delete facial data: %w", err)
		}
	}

	// Delete the user
	_, err = tx.Exec("DELETE FROM users WHERE id = ?", userID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete user: %w", err)
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
