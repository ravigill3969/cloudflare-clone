package ddos

import (
	"fmt"
	"sync"
	"time"
)

type TokenBucket struct {
	tokens chan struct{}
}

func NewTokenBucket(capacity int, refillInterval time.Duration) *TokenBucket {
	b := &TokenBucket{
		tokens: make(chan struct{}, capacity),
	}

	// Fill initially
	for i := 0; i < capacity; i++ {
		b.tokens <- struct{}{}
	}


	go func() {
		ticker := time.NewTicker(refillInterval)
		defer ticker.Stop()

		for range ticker.C {
			select {
			case b.tokens <- struct{}{}:
				fmt.Println("Token added")
			default:
				fmt.Println("bucket full")
			}
		}
	}()

	return b
}

func (b *TokenBucket) Take() {
	<-b.tokens
}

type DDoSLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*TokenBucket
	cap      int
	interval time.Duration
}

func NewDDoSLimiter(cap int, interval time.Duration) *DDoSLimiter {
	return &DDoSLimiter{
		buckets:  make(map[string]*TokenBucket),
		cap:      cap,
		interval: interval,
	}
}

func (l *DDoSLimiter) getBucket(ip string) *TokenBucket {
	l.mu.Lock()
	defer l.mu.Unlock()

	if bucket, ok := l.buckets[ip]; ok {
		return bucket
	}
	bucket := NewTokenBucket(l.cap, l.interval)
	l.buckets[ip] = bucket
	return bucket
}

// AllowRequest blocks until a token is available for this IP
func (l *DDoSLimiter) AllowRequest(ip string) {
	b := l.getBucket(ip)
	b.Take()
}
