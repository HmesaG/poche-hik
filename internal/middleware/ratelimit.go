package middleware

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter middleware for rate limiting
type RateLimiter struct {
	visitors map[string]*visitor
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewRateLimiter creates a new rate limiter middleware
func NewRateLimiter(requestsPerSecond int, burst int) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate.Limit(requestsPerSecond),
		burst:    burst,
	}

	// Cleanup old visitors every minute
	go rl.cleanup()

	return rl
}

// Middleware returns the HTTP middleware for rate limiting
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get client IP
		ip := getClientIP(r)

		// Get or create visitor
		limiter := rl.getLimiter(ip)

		// Check if allowed
		if !limiter.Allow() {
			http.Error(w, `{"error": "Too many requests. Please try again later."}`, http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// getLimiter returns or creates a rate limiter for a visitor
func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(rl.rate, rl.burst)
		v = &visitor{limiter: limiter, lastSeen: time.Now()}
		rl.visitors[ip] = v
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

// cleanup removes visitors that haven't been seen in 3 minutes
func (rl *RateLimiter) cleanup() {
	for {
		time.Sleep(time.Minute)
		rl.mu.Lock()
		for ip, visitor := range rl.visitors {
			if time.Since(visitor.lastSeen) > 3*time.Minute {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		return xff
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if ip == "" {
		return "unknown"
	}

	return ip
}

// AuthRateLimiter is a stricter rate limiter for auth endpoints
type AuthRateLimiter struct {
	attempts map[string]*authAttempt
	mu       sync.RWMutex
	maxAttempts int
	window     time.Duration
}

type authAttempt struct {
	count    int
	lastTime time.Time
}

// NewAuthRateLimiter creates a rate limiter for authentication endpoints
func NewAuthRateLimiter(maxAttempts int, window time.Duration) *AuthRateLimiter {
	arl := &AuthRateLimiter{
		attempts:    make(map[string]*authAttempt),
		maxAttempts: maxAttempts,
		window:      window,
	}

	// Cleanup old attempts every minute
	go arl.cleanup()

	return arl
}

// Middleware returns the HTTP middleware for auth rate limiting
func (arl *AuthRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)

		if !arl.allow(ip) {
			http.Error(w, `{"error": "Too many login attempts. Please try again later."}`, http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// allow checks if an IP is allowed to make a request
func (arl *AuthRateLimiter) allow(ip string) bool {
	arl.mu.Lock()
	defer arl.mu.Unlock()

	attempt, exists := arl.attempts[ip]
	if !exists {
		arl.attempts[ip] = &authAttempt{count: 1, lastTime: time.Now()}
		return true
	}

	// Reset if window has passed
	if time.Since(attempt.lastTime) > arl.window {
		attempt.count = 1
		attempt.lastTime = time.Now()
		return true
	}

	// Check if max attempts reached
	if attempt.count >= arl.maxAttempts {
		return false
	}

	attempt.count++
	attempt.lastTime = time.Now()
	return true
}

// RecordFailure records a failed login attempt
func (arl *AuthRateLimiter) RecordFailure(ip string) {
	arl.mu.Lock()
	defer arl.mu.Unlock()

	attempt, exists := arl.attempts[ip]
	if !exists {
		arl.attempts[ip] = &authAttempt{count: 1, lastTime: time.Now()}
		return
	}

	attempt.count++
	attempt.lastTime = time.Now()
}

// cleanup removes old attempts
func (arl *AuthRateLimiter) cleanup() {
	for {
		time.Sleep(time.Minute)
		arl.mu.Lock()
		for ip, attempt := range arl.attempts {
			if time.Since(attempt.lastTime) > arl.window {
				delete(arl.attempts, ip)
			}
		}
		arl.mu.Unlock()
	}
}
