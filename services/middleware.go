package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Session storage
type Session struct {
	ID        string
	UserID    uuid.UUID
	Username  string
	Email     string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type SessionManager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
	timeout  time.Duration
}

var sessionManager = &SessionManager{
	sessions: make(map[string]*Session),
	timeout:  24 * time.Hour, // 24 hour session timeout
}

// Initialize session cleanup
func init() {
	go sessionManager.cleanupExpiredSessions()
}

// Generate secure session ID
func generateSessionID() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// CreateSession - Creates a new session for authenticated user
func (sm *SessionManager) CreateSession(userID uuid.UUID, username, email string) (string, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return "", err
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	session := &Session{
		ID:        sessionID,
		UserID:    userID,
		Username:  username,
		Email:     email,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(sm.timeout),
	}

	sm.sessions[sessionID] = session
	log.Printf("Session created for user %s (ID: %s)", username, sessionID)
	return sessionID, nil
}

// GetSession - Retrieves session by ID
func (sm *SessionManager) GetSession(sessionID string) (*Session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, false
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		return nil, false
	}

	return session, true
}

// DeleteSession - Removes a session
func (sm *SessionManager) DeleteSession(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, sessionID)
	log.Printf("Session deleted: %s", sessionID)
}

// RefreshSession - Extends session expiration
func (sm *SessionManager) RefreshSession(sessionID string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return false
	}

	session.ExpiresAt = time.Now().Add(sm.timeout)
	return true
}

// Cleanup expired sessions periodically
func (sm *SessionManager) cleanupExpiredSessions() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		sm.mu.Lock()
		now := time.Now()
		for id, session := range sm.sessions {
			if now.After(session.ExpiresAt) {
				delete(sm.sessions, id)
				log.Printf("Expired session cleaned up: %s", id)
			}
		}
		sm.mu.Unlock()
	}
}

// SessionAuth Middleware - Verifies session and adds user info to context
func SessionAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get session ID from header or cookie
		sessionID := r.Header.Get("X-Session-ID")
		if sessionID == "" {
			// Try to get from cookie as fallback
			cookie, err := r.Cookie("session_id")
			if err == nil {
				sessionID = cookie.Value
			}
		}

		if sessionID == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Session ID required. Please login first.",
			})
			return
		}

		// Validate session
		session, exists := sessionManager.GetSession(sessionID)
		if !exists {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Invalid or expired session. Please login again.",
			})
			return
		}

		// Refresh session on each valid request
		sessionManager.RefreshSession(sessionID)

		// Add session info to context
		ctx := context.WithValue(r.Context(), "session", session)
		ctx = context.WithValue(ctx, "user_id", session.UserID)
		ctx = context.WithValue(ctx, "username", session.Username)
		ctx = context.WithValue(ctx, "email", session.Email)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetSessionFromContext - Helper to extract session from request context
func GetSessionFromContext(r *http.Request) (*Session, bool) {
	session, ok := r.Context().Value("session").(*Session)
	return session, ok
}

// CORS Middleware
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
		if allowedOrigins == "" {
			allowedOrigins = "*"
		}

		w.Header().Set("Access-Control-Allow-Origin", allowedOrigins)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-ID, X-Project-ID, X-Session-ID")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Expose-Headers", "X-Session-ID")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Request Logger Middleware
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)
		duration := time.Since(start)
		log.Printf("%s %s %s %d %v %s", r.RemoteAddr, r.Method, r.RequestURI, wrapped.statusCode, duration, r.UserAgent())
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Rate Limiter using token bucket algorithm
type RateLimiter struct {
	visitors map[string]*visitor
	mu       sync.RWMutex
	rate     int
	burst    int
	cleanup  time.Duration
}

type visitor struct {
	tokens     int
	lastRefill time.Time
	mu         sync.Mutex
}

func NewRateLimiter(requestsPerMinute, burst int) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     requestsPerMinute,
		burst:    burst,
		cleanup:  5 * time.Minute,
	}
	go rl.cleanupVisitors()
	return rl
}

func (rl *RateLimiter) getVisitor(ip string) *visitor {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	if !exists {
		v = &visitor{
			tokens:     rl.burst,
			lastRefill: time.Now(),
		}
		rl.visitors[ip] = v
	}
	return v
}

func (rl *RateLimiter) cleanupVisitors() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastRefill) > rl.cleanup {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func (v *visitor) allow(rate, burst int) bool {
	v.mu.Lock()
	defer v.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(v.lastRefill)

	tokensToAdd := int(elapsed.Seconds() * float64(rate) / 60.0)
	if tokensToAdd > 0 {
		v.tokens += tokensToAdd
		if v.tokens > burst {
			v.tokens = burst
		}
		v.lastRefill = now
	}

	if v.tokens > 0 {
		v.tokens--
		return true
	}

	return false
}

func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			ip = strings.Split(forwarded, ",")[0]
		}

		visitor := rl.getVisitor(ip)
		if !visitor.allow(rl.rate, rl.burst) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Rate limit exceeded. Please try again later.",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

// User Context Middleware
func UserContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-User-ID")
		if userID != "" {
			if _, err := uuid.Parse(userID); err == nil {
				ctx := context.WithValue(r.Context(), "user_id", userID)
				r = r.WithContext(ctx)
			}
		}
		next.ServeHTTP(w, r)
	})
}

// Project Context Middleware
func ProjectContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		projectID := r.Header.Get("X-Project-ID")
		if projectID != "" {
			if _, err := uuid.Parse(projectID); err == nil {
				ctx := context.WithValue(r.Context(), "project_id", projectID)
				r = r.WithContext(ctx)
			}
		}
		next.ServeHTTP(w, r)
	})
}

// Recovery Middleware
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("PANIC: %v", err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "Internal server error",
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Content Type Middleware
func JSONContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

// Security Headers Middleware
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}
