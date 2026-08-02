package controlplane

import (
	"context"
	"sync"
	"time"
)

// TelegramRateLimiter controls outbound Telegram Bot API requests based on official rate limits.
type TelegramRateLimiter struct {
	mu           sync.Mutex
	globalTokens float64
	globalLast   time.Time

	chatTokens map[int64]float64
	chatLast   map[int64]time.Time
}

// NewTelegramRateLimiter creates a thread-safe Telegram Bot rate limiter.
func NewTelegramRateLimiter() *TelegramRateLimiter {
	return &TelegramRateLimiter{
		globalTokens: 30.0,
		globalLast:   time.Now(),
		chatTokens:   make(map[int64]float64),
		chatLast:     make(map[int64]time.Time),
	}
}

// Wait blocks until both the global limit (30 rps) and the chat-specific limit (1 rps for private, 20/min for groups) are satisfied.
func (l *TelegramRateLimiter) Wait(ctx context.Context, chatID int64, isGroup bool) error {
	for {
		l.mu.Lock()
		now := time.Now()

		// Global token bucket (replenish at 30/sec, max 30)
		elapsedGlobal := now.Sub(l.globalLast).Seconds()
		l.globalTokens += elapsedGlobal * 30.0
		if l.globalTokens > 30.0 {
			l.globalTokens = 30.0
		}
		l.globalLast = now

		// Chat token bucket
		rate := 1.0 // 1 req/sec for private chats
		maxTokens := 1.0
		if isGroup {
			rate = 20.0 / 60.0 // 20 req/min for group/channels
			maxTokens = 20.0
		}

		cTokens := l.chatTokens[chatID]
		cLast, hasLast := l.chatLast[chatID]
		if !hasLast {
			cTokens = maxTokens
		} else {
			elapsedChat := now.Sub(cLast).Seconds()
			cTokens += elapsedChat * rate
			if cTokens > maxTokens {
				cTokens = maxTokens
			}
		}
		l.chatTokens[chatID] = cTokens

		if l.globalTokens >= 1.0 && cTokens >= 1.0 {
			l.globalTokens -= 1.0
			l.chatTokens[chatID] = cTokens - 1.0
			l.chatLast[chatID] = now
			l.mu.Unlock()
			return nil
		}

		// Calculate wait time
		var waitTime time.Duration
		if l.globalTokens < 1.0 {
			neededGlobal := (1.0 - l.globalTokens) / 30.0
			waitTime = time.Duration(neededGlobal * float64(time.Second))
		}
		if cTokens < 1.0 {
			neededChat := (1.0 - cTokens) / rate
			chatWait := time.Duration(neededChat * float64(time.Second))
			if chatWait > waitTime {
				waitTime = chatWait
			}
		}
		l.mu.Unlock()

		if waitTime <= 0 {
			waitTime = 10 * time.Millisecond
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
		}
	}
}

// BackoffChat forces a specific chat to wait for the duration specified by Telegram.
func (l *TelegramRateLimiter) BackoffChat(chatID int64, retryAfter time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	rate := 1.0
	l.chatTokens[chatID] = -float64(retryAfter.Seconds()) * rate
	l.chatLast[chatID] = time.Now()
}

// BackoffGlobal forces all outbound requests to backoff for the duration specified by Telegram.
func (l *TelegramRateLimiter) BackoffGlobal(retryAfter time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.globalTokens = -float64(retryAfter.Seconds()) * 30.0
	l.globalLast = time.Now()
}
