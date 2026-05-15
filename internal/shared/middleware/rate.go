package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	clients = make(map[string]*client)
	mu      sync.RWMutex
)

func RateLimiterMiddleware(lmt rate.Limit, burst int) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		ip = strings.Split(ip, ":")[0]
		if ip == "" {
			ip = "127.0.0.1"
		}

		mu.RLock()
		cl, ok := clients[ip]
		mu.RUnlock()

		if !ok {
			mu.Lock()
			cl, ok = clients[ip]
			if !ok {
				cl = &client{
					limiter:  rate.NewLimiter(lmt, burst),
					lastSeen: time.Now(),
				}
				clients[ip] = cl
			}
			mu.Unlock()
		}

		passed := cl.limiter.Allow()
		if passed {
			cl.lastSeen = time.Now()
		}

		if !passed {
			delay := cl.limiter.Reserve().DelayFrom(time.Now()).Seconds()
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"retry_after": delay,
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

func StartCleanup() {
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			mu.Lock()
			now := time.Now()
			for ip, cl := range clients {
				if now.Sub(cl.lastSeen) > time.Hour {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()
}
