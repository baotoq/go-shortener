package http

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"go-shortener/pkg/problemdetails"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// entry holds a rate limiter and last seen timestamp for cleanup
type entry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter provides per-IP rate limiting
type RateLimiter struct {
	limiters  map[string]*entry
	mu        sync.RWMutex
	rateLimit rate.Limit
	burst     int
}

// NewRateLimiter creates a new rate limiter with the given requests per minute
func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	rl := &RateLimiter{
		limiters:  make(map[string]*entry),
		rateLimit: rate.Every(time.Minute / time.Duration(requestsPerMinute)),
		burst:     requestsPerMinute,
	}
	rl.StartCleanup()
	return rl
}

// getLimiter returns the rate limiter for the given IP, creating one if it doesn't exist
func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	e, exists := rl.limiters[ip]
	if !exists {
		limiter := rate.NewLimiter(rl.rateLimit, rl.burst)
		rl.limiters[ip] = &entry{
			limiter:  limiter,
			lastSeen: time.Now(),
		}
		return limiter
	}

	e.lastSeen = time.Now()
	return e.limiter
}

// Middleware returns a middleware that enforces rate limiting
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get client IP (RealIP middleware runs before this)
		ip := r.RemoteAddr

		limiter := rl.getLimiter(ip)

		if !limiter.Allow() {
			// Rate limit exceeded
			resetTime := time.Now().Add(time.Minute).Unix()
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rl.burst))
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime))

			problem := problemdetails.New(
				http.StatusTooManyRequests,
				problemdetails.TypeRateLimitExceeded,
				"Rate Limit Exceeded",
				"Too many requests. Please try again later.",
			)
			writeProblem(w, problem)
			return
		}

		// Set rate limit headers
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rl.burst))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", int(limiter.Tokens())))

		next.ServeHTTP(w, r)
	})
}

// StartCleanup starts a background goroutine that cleans up old entries
func (rl *RateLimiter) StartCleanup() {
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			rl.mu.Lock()
			for ip, e := range rl.limiters {
				if time.Since(e.lastSeen) > time.Hour {
					delete(rl.limiters, ip)
				}
			}
			rl.mu.Unlock()
		}
	}()
}

// LoggerMiddleware returns a middleware that logs HTTP requests using Zap
func LoggerMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				logger.Info("http request",
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.Int("status", ww.Status()),
					zap.Duration("duration", time.Since(start)),
					zap.String("remote_addr", r.RemoteAddr),
					zap.String("request_id", middleware.GetReqID(r.Context())),
				)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}
