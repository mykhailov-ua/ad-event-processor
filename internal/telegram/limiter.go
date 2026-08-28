package telegram

import (
	"context"
	"sync"
	"time"
)

type RateLimiter struct {
	mu           sync.Mutex
	globalTokens float64
	globalLast   time.Time

	chatTokens map[int64]float64
	chatLast   map[int64]time.Time
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		globalTokens: 30.0,
		globalLast:   time.Now(),
		chatTokens:   make(map[int64]float64),
		chatLast:     make(map[int64]time.Time),
	}
}

func (l *RateLimiter) Wait(ctx context.Context, chatID int64, isGroup bool) error {
	for {
		l.mu.Lock()
		now := time.Now()

		elapsedGlobal := now.Sub(l.globalLast).Seconds()
		l.globalTokens += elapsedGlobal * 30.0
		if l.globalTokens > 30.0 {
			l.globalTokens = 30.0
		}
		l.globalLast = now

		rate := 1.0
		maxTokens := 1.0
		if isGroup {
			rate = 20.0 / 60.0
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

func (l *RateLimiter) BackoffChat(chatID int64, retryAfter time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	rate := 1.0
	l.chatTokens[chatID] = -float64(retryAfter.Seconds()) * rate
	l.chatLast[chatID] = time.Now()
}

func (l *RateLimiter) BackoffGlobal(retryAfter time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.globalTokens = -float64(retryAfter.Seconds()) * 30.0
	l.globalLast = time.Now()
}
