package middleware

import (
	"net/http"

	"golang.org/x/time/rate"
)

func RateLimit(rps rate.Limit, burst int) func(http.Handler) http.Handler {
	limiter := rate.NewLimiter(rps, burst)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow() {
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func PerIPRateLimit(rps rate.Limit, burst int) func(http.Handler) http.Handler {
	visitors := make(map[string]*rate.Limiter)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr

			limiter, exists := visitors[ip]
			if !exists {
				limiter = rate.NewLimiter(rps, burst)
				visitors[ip] = limiter
			}

			if !limiter.Allow() {
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
