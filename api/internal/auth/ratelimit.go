package auth

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/thoutmose/axon/api/internal/httpx"
)

// limiter allows each client IP a small burst of sign-in attempts, refilled
// slowly. Axon runs as one API process, so in-memory state is enough.
type limiter struct {
	mu      sync.Mutex
	clients map[string]*rate.Limiter
	every   time.Duration
	burst   int
}

func newLimiter(every time.Duration, burst int) *limiter {
	return &limiter{clients: map[string]*rate.Limiter{}, every: every, burst: burst}
}

func (l *limiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.clients) > 10000 { // bounded memory: forget everyone, rarely
		l.clients = map[string]*rate.Limiter{}
	}
	c, ok := l.clients[ip]
	if !ok {
		c = rate.NewLimiter(rate.Every(l.every), l.burst)
		l.clients[ip] = c
	}
	return c.Allow()
}

func (l *limiter) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.allow(httpx.ClientIP(r)) {
			w.Header().Set("Retry-After", "60")
			httpx.Problem(w, http.StatusTooManyRequests, "Too many sign-in attempts. Wait a minute and try again.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Throttle applies the sign-in limit to other routes that take a secret
// from an anonymous caller, such as a vault pairing code.
func (s *Service) Throttle(next http.Handler) http.Handler { return s.signInLimit.middleware(next) }
