package middleware

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

func PerIPRateLimit(rps rate.Limit, burst int) func(http.Handler) http.Handler {
	var mu sync.Mutex
	visitors := make(map[string]*ipEntry)

	go func() {
		for {
			time.Sleep(time.Minute)
			mu.Lock()
			for ip, e := range visitors {
				if time.Since(e.lastSeen) > 3*time.Minute {
					delete(visitors, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr

			mu.Lock()
			entry, exists := visitors[ip]
			if !exists {
				entry = &ipEntry{limiter: rate.NewLimiter(rps, burst)}
				visitors[ip] = entry
			}
			entry.lastSeen = time.Now()
			limiter := entry.limiter
			mu.Unlock()

			if !limiter.Allow() {
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

type ipEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}
