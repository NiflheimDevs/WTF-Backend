package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type RateLimiter struct {
	mu       sync.Mutex
	clients  map[string]*clientLimiter
	rate     rate.Limit
	burst    int
	lastSeen time.Duration
}

type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func NewRateLimiter(requestsPerMinute, burst int) *RateLimiter {
	if requestsPerMinute < 1 {
		requestsPerMinute = 10
	}
	if burst < 1 {
		burst = requestsPerMinute
	}
	return &RateLimiter{
		clients:  make(map[string]*clientLimiter),
		rate:     rate.Every(time.Minute / time.Duration(requestsPerMinute)),
		burst:    burst,
		lastSeen: 10 * time.Minute,
	}
}

func (r *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		limiter := r.limiterFor(clientIP(req))
		if !limiter.Allow() {
			writeError(w, http.StatusTooManyRequests, "rate_limited", "too many requests from this IP")
			return
		}
		next.ServeHTTP(w, req)
	})
}

func (r *RateLimiter) limiterFor(ip string) *rate.Limiter {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	for key, client := range r.clients {
		if now.Sub(client.lastSeen) > r.lastSeen {
			delete(r.clients, key)
		}
	}

	client, ok := r.clients[ip]
	if !ok {
		client = &clientLimiter{limiter: rate.NewLimiter(r.rate, r.burst)}
		r.clients[ip] = client
	}
	client.lastSeen = now
	return client.limiter
}

func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
