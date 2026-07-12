package auth

import (
	"sync"
	"time"
)

type attemptInfo struct {
	count       int
	lockedUntil time.Time
}

type LoginRateLimiter struct {
	mu              sync.Mutex
	attempts        map[string]*attemptInfo
	maxAttempts     int
	lockoutDuration time.Duration
}

func NewLoginRateLimiter(maxAttempts int, lockoutDuration time.Duration) *LoginRateLimiter {
	return &LoginRateLimiter{
		attempts:        make(map[string]*attemptInfo),
		maxAttempts:     maxAttempts,
		lockoutDuration: lockoutDuration,
	}
}

func (rl *LoginRateLimiter) IsLocked(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	attempt, exists := rl.attempts[key]
	if !exists {
		return false
	}

	return time.Now().Before(attempt.lockedUntil)
}

func (rl *LoginRateLimiter) RecordFailure(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	attempt, exists := rl.attempts[key]
	if !exists {
		attempt = &attemptInfo{}
		rl.attempts[key] = attempt
	}

	attempt.count++
	if attempt.count >= rl.maxAttempts {
		attempt.lockedUntil = time.Now().Add(rl.lockoutDuration)
		attempt.count = 0
	}
}

func (rl *LoginRateLimiter) RecordSuccess(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.attempts, key)
}
