package main

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// app/ROADMAP.md Phase 3 gap fix: no rate-limiting/DoS-protection middleware
// existed anywhere in prototype/go's own handler layer (Study 34 §5) --
// closed here, while the middleware stack is being wired fresh anyway, the
// same "cheapest moment" reasoning the migrations/API-versioning gaps above
// already used. A per-IP token bucket, hand-rolled like every other
// middleware in this file (sessionAuth/csrfProtect/workspaceTx) rather than
// reaching for a third-party middleware framework -- golang.org/x/time/rate
// is the one small, official extended-stdlib package this needs.
//
// Defaults, calibrated against real traffic once app/ finally had some:
// app/ROADMAP.md Phase 4's own first full conformance run against these
// original, ungrounded numbers (10 req/s, burst 30) failed 98/219 tests,
// the overwhelming majority on an unexpected 429 -- one client (the suite
// itself, all requests genuinely from one IP in this dev setup) peaked at
// 57 requests in a single wall-clock second, a legitimate-traffic shape
// this limiter must not choke on (a real browser session issuing several
// HTMX/typeahead/asset requests in quick succession looks the same). Reset
// to 30 req/s sustained / burst 120 -- comfortably above that measured
// peak with real headroom, while still capping a genuine sustained flood
// from one IP far below what this single-vCPU host could be pushed to.
// Revisit again if a real deployment's own traffic shape says otherwise.
const (
	rateLimitPerSecond = 30
	rateLimitBurst     = 120
	rateLimitIdleTTL   = 10 * time.Minute
	rateLimitSweep     = 5 * time.Minute
)

// ipRateLimiter tracks one token bucket per client IP, evicting entries
// idle longer than rateLimitIdleTTL so a long-running process doesn't
// accumulate one *rate.Limiter forever per distinct IP that has ever
// connected (this host runs a single process, no external store to offload
// this to -- see ARCHITECTURE.md's "Deployment target: no containers").
type ipRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rateLimiterEntry
}

type rateLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func newIPRateLimiter() *ipRateLimiter {
	l := &ipRateLimiter{limiters: make(map[string]*rateLimiterEntry)}
	go l.sweepLoop()
	return l
}

func (l *ipRateLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.limiters[ip]
	if !ok {
		e = &rateLimiterEntry{limiter: rate.NewLimiter(rateLimitPerSecond, rateLimitBurst)}
		l.limiters[ip] = e
	}
	e.lastSeen = time.Now()
	return e.limiter.Allow()
}

func (l *ipRateLimiter) sweepLoop() {
	ticker := time.NewTicker(rateLimitSweep)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-rateLimitIdleTTL)
		l.mu.Lock()
		for ip, e := range l.limiters {
			if e.lastSeen.Before(cutoff) {
				delete(l.limiters, ip)
			}
		}
		l.mu.Unlock()
	}
}

// rateLimit rejects a request over its client IP's own budget with 429 --
// keyed on r.RemoteAddr, which by this point in the middleware chain has
// already been corrected by chi's middleware.RealIP (must run before this
// one, see main()) from Caddy's own X-Real-IP header; without that, every
// request would appear to come from Caddy's loopback address, and this
// would rate-limit all traffic together instead of per real client.
func (l *ipRateLimiter) rateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr // no port present -- use as-is rather than reject
		}
		if !l.allow(host) {
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
