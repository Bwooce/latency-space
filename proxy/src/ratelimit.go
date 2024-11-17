package main

import (
	"golang.org/x/time/rate"
	"sync"
)

type IPRateLimiter struct {
	ips map[string]*rate.Limiter
	mu  *sync.RWMutex
	r   rate.Limit
	b   int
}

func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	return &IPRateLimiter{
		ips: make(map[string]*rate.Limiter),
		mu:  &sync.RWMutex{},
		r:   r,
		b:   b,
	}
}

func (l *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	limiter, exists := l.ips[ip]
	if !exists {
		limiter = rate.NewLimiter(l.r, l.b)
		l.ips[ip] = limiter
	}

	return limiter
}

var (
	tcpLimiters = make(map[string]*IPRateLimiter)
	udpLimiters = make(map[string]*IPRateLimiter)
)

func initRateLimiters() {
	// Initialize limiters for all celestial bodies
	for name, body := range solarSystem {
		tcpLimiters[name] = NewIPRateLimiter(rate.Limit(body.RateLimit/60), body.RateLimit/10)
		udpLimiters[name] = NewIPRateLimiter(rate.Limit(body.RateLimit/60), body.RateLimit/10)

		for moonName, moon := range body.Moons {
			fullName := moonName + "." + name
			tcpLimiters[fullName] = NewIPRateLimiter(rate.Limit(moon.RateLimit/60), moon.RateLimit/10)
			udpLimiters[fullName] = NewIPRateLimiter(rate.Limit(moon.RateLimit/60), moon.RateLimit/10)
		}
	}

	for name, craft := range spacecraft {
		tcpLimiters[name] = NewIPRateLimiter(rate.Limit(craft.RateLimit/60), craft.RateLimit/10)
		udpLimiters[name] = NewIPRateLimiter(rate.Limit(craft.RateLimit/60), craft.RateLimit/10)
	}
}
