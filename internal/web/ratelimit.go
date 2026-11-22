package web

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// rateLimiter is an in-memory token bucket per key (the client address). It
// suits a single instance. Behind several instances, enforce limits at the
// reverse proxy or back this with a shared store.
type rateLimiter struct {
	limit rate.Limit
	burst int
	ttl   time.Duration

	mu        sync.Mutex
	visitors  map[string]*visitor
	lastSweep time.Time
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// newRateLimiter allows bursts of up to events, refilling at events per
// window.
func newRateLimiter(events int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		limit:     rate.Every(window / time.Duration(events)),
		burst:     events,
		ttl:       2 * window,
		visitors:  make(map[string]*visitor),
		lastSweep: time.Now(),
	}
}

func (l *rateLimiter) allow(key string) bool {
	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	// Forget idle visitors so the map cannot grow without bound. A visitor
	// idle for the TTL has a full bucket again, so dropping it loses nothing.
	if now.Sub(l.lastSweep) > l.ttl {
		for k, v := range l.visitors {
			if now.Sub(v.lastSeen) > l.ttl {
				delete(l.visitors, k)
			}
		}
		l.lastSweep = now
	}

	v, ok := l.visitors[key]
	if !ok {
		v = &visitor{limiter: rate.NewLimiter(l.limit, l.burst)}
		l.visitors[key] = v
	}
	v.lastSeen = now

	return v.limiter.AllowN(now, 1)
}
