package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// SessionState represents the current session state
type SessionState struct {
	SessionID   string    `json:"session_id"`
	CreatedAt   time.Time `json:"created_at"`
	LastUsed    time.Time `json:"last_used"`
	LoopCount   int       `json:"loop_count"`
	ResetAt     time.Time `json:"reset_at,omitempty"`
	ResetReason string    `json:"reset_reason,omitempty"`
}

// SessionManager handles session persistence and expiration
type SessionManager struct {
	sessionFile string
	historyFile string
	expiryHours int
}

// NewSessionManager creates a session manager
func NewSessionManager(sessionFile, historyFile string, expiryHours int) *SessionManager {
	if expiryHours <= 0 {
		expiryHours = 24 // Default 24 hour expiry
	}
	return &SessionManager{
		sessionFile: sessionFile,
		historyFile: historyFile,
		expiryHours: expiryHours,
	}
}

// Load reads the current session state
func (m *SessionManager) Load() (*SessionState, error) {
	data, err := os.ReadFile(m.sessionFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No session exists
		}
		return nil, err
	}

	var state SessionState
	if err := json.Unmarshal(data, &state); err != nil {
		// Corrupted or incompatible session file - treat as no session
		return nil, nil
	}

	// Check for invalid/empty timestamps (from bash version)
	if state.CreatedAt.IsZero() || state.LastUsed.IsZero() {
		return nil, nil // Invalid session, create new
	}

	return &state, nil
}

// Save writes the session state
func (m *SessionManager) Save(state *SessionState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.sessionFile, data, 0644)
}

// GetOrCreate retrieves existing session or creates a new one
func (m *SessionManager) GetOrCreate() (*SessionState, bool, error) {
	state, err := m.Load()
	if err != nil {
		return nil, false, err
	}

	// Check if session exists and is not expired
	if state != nil && !m.isExpired(state) {
		state.LastUsed = time.Now()
		state.LoopCount++
		if err := m.Save(state); err != nil {
			return nil, false, err
		}
		return state, false, nil // Existing session
	}

	// Create new session
	newState := &SessionState{
		SessionID: generateSessionID(),
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
		LoopCount: 1,
	}

	if err := m.Save(newState); err != nil {
		return nil, false, err
	}

	// Log transition if there was a previous session
	if state != nil {
		m.logTransition("active", "expired", "session_expired", state.LoopCount)
	}

	return newState, true, nil // New session
}

// isExpired checks if a session has expired
func (m *SessionManager) isExpired(state *SessionState) bool {
	expiryDuration := time.Duration(m.expiryHours) * time.Hour
	return time.Since(state.LastUsed) > expiryDuration
}

// Reset clears the session with a reason
func (m *SessionManager) Reset(reason string) error {
	state, _ := m.Load()
	loopCount := 0
	if state != nil {
		loopCount = state.LoopCount
	}

	newState := &SessionState{
		SessionID:   generateSessionID(),
		CreatedAt:   time.Now(),
		LastUsed:    time.Now(),
		LoopCount:   0,
		ResetAt:     time.Now(),
		ResetReason: reason,
	}

	if err := m.Save(newState); err != nil {
		return err
	}

	m.logTransition("active", "reset", reason, loopCount)
	return nil
}

// SessionTransition represents a recorded state change
type SessionTransition struct {
	Timestamp time.Time `json:"timestamp"`
	FromState string    `json:"from_state"`
	ToState   string    `json:"to_state"`
	Reason    string    `json:"reason"`
	LoopCount int       `json:"loop_count"`
}

// logTransition records a session state change
func (m *SessionManager) logTransition(from, to, reason string, loopCount int) {
	transition := SessionTransition{
		Timestamp: time.Now(),
		FromState: from,
		ToState:   to,
		Reason:    reason,
		LoopCount: loopCount,
	}

	// Load existing history
	var history []SessionTransition
	if data, err := os.ReadFile(m.historyFile); err == nil {
		json.Unmarshal(data, &history)
	}

	// Append and keep only last 50
	history = append(history, transition)
	if len(history) > 50 {
		history = history[len(history)-50:]
	}

	// Save history
	if data, err := json.MarshalIndent(history, "", "  "); err == nil {
		os.WriteFile(m.historyFile, data, 0644)
	}
}

// generateSessionID creates a unique session identifier
func generateSessionID() string {
	return fmt.Sprintf("ralph-%d-%d", time.Now().Unix(), os.Getpid())
}
