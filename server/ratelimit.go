package server

import (
	"sync"
	"time"
)

const (
	defaultAuthFailureLimit    = 5
	defaultAuthFailureWindow   = time.Minute
	defaultAuthBlockDuration   = 10 * time.Minute
	defaultAuthCleanupInterval = time.Minute
)

// failedAuthLimiter tracks failed authentication attempts per client and
// blocks clients that exceed the failure threshold.
type failedAuthLimiter struct {
	mu              sync.Mutex
	attempts        map[string]failedAuthAttempt
	failureLimit    int
	window          time.Duration
	blockDuration   time.Duration
	cleanupInterval time.Duration
	entryTTL        time.Duration
	lastCleanup     time.Time
}

type failedAuthAttempt struct {
	windowStart  time.Time
	failures     int
	blockedUntil time.Time
	lastSeen     time.Time
}

func newFailedAuthLimiter(failureLimit int, window time.Duration, blockDuration time.Duration) *failedAuthLimiter {
	return &failedAuthLimiter{
		attempts:        make(map[string]failedAuthAttempt),
		failureLimit:    failureLimit,
		window:          window,
		blockDuration:   blockDuration,
		cleanupInterval: defaultAuthCleanupInterval,
		entryTTL:        window + blockDuration,
		lastCleanup:     time.Now(),
	}
}

func (l *failedAuthLimiter) Blocked(clientID string, now time.Time) (time.Time, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.cleanupLocked(now)
	attempt := l.attempts[clientID]
	blockedUntil, blocked := l.resolveBlockLocked(&attempt, now)
	l.attempts[clientID] = attempt
	return blockedUntil, blocked
}

func (l *failedAuthLimiter) CheckAndRecord(clientID string, now time.Time) (time.Time, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.cleanupLocked(now)
	attempt := l.attempts[clientID]

	if blockedUntil, blocked := l.resolveBlockLocked(&attempt, now); blocked {
		l.attempts[clientID] = attempt
		return blockedUntil, true
	}

	if attempt.windowStart.IsZero() || now.Sub(attempt.windowStart) > l.window {
		attempt.windowStart = now
		attempt.failures = 0
	}
	attempt.failures++
	attempt.lastSeen = now
	if attempt.failures >= l.failureLimit {
		attempt.blockedUntil = now.Add(l.blockDuration)
	}
	l.attempts[clientID] = attempt
	return attempt.blockedUntil, attempt.blockedUntil.After(now)
}

// resolveBlockLocked checks if the attempt is currently blocked. If the block
// has expired, it resets the attempt state. Must be called with mu held.
func (l *failedAuthLimiter) resolveBlockLocked(attempt *failedAuthAttempt, now time.Time) (time.Time, bool) {
	if attempt.blockedUntil.After(now) {
		attempt.lastSeen = now
		return attempt.blockedUntil, true
	}
	if !attempt.blockedUntil.IsZero() {
		attempt.blockedUntil = time.Time{}
		attempt.failures = 0
		attempt.windowStart = time.Time{}
		attempt.lastSeen = now
	}
	return time.Time{}, false
}

func (l *failedAuthLimiter) cleanupLocked(now time.Time) {
	if now.Sub(l.lastCleanup) < l.cleanupInterval {
		return
	}
	for clientID, attempt := range l.attempts {
		if now.Sub(attempt.lastSeen) > l.entryTTL {
			delete(l.attempts, clientID)
		}
	}
	l.lastCleanup = now
}
