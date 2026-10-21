package app

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	maxLoginClients = 1024
	loginClientIdle = 15 * time.Minute
)

type loginVisitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type loginLimiter struct {
	mu      sync.Mutex
	clients map[netip.Addr]*loginVisitor
}

func newLoginLimiter() *loginLimiter {
	return &loginLimiter{clients: make(map[netip.Addr]*loginVisitor)}
}

func (l *loginLimiter) allow(client netip.Addr, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	visitor := l.clients[client]
	if visitor == nil {
		var (
			oldest     netip.Addr
			oldestTime time.Time
		)
		for address, existing := range l.clients {
			if now.Sub(existing.lastSeen) >= loginClientIdle {
				delete(l.clients, address)
				continue
			}

			if oldestTime.IsZero() || existing.lastSeen.Before(oldestTime) {
				oldest, oldestTime = address, existing.lastSeen
			}
		}

		// Evict rather than deny a new client when the bounded cache is full.
		if len(l.clients) >= maxLoginClients {
			delete(l.clients, oldest)
		}

		visitor = &loginVisitor{limiter: rate.NewLimiter(rate.Every(12*time.Second), 5)}
		l.clients[client] = visitor
	}

	visitor.lastSeen = now

	return visitor.limiter.AllowN(now, 1)
}

func (s *Server) trustedProxy(address netip.Addr) bool {
	for _, prefix := range s.config.TrustedProxies {
		if prefix.Contains(address) {
			return true
		}
	}

	return false
}

func (s *Server) clientIP(r *http.Request) netip.Addr {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	peer, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}
	}

	peer = peer.Unmap()
	if !s.trustedProxy(peer) {
		return peer
	}

	// Walk from the trusted peer toward the client; ignore spoofed hops to its left.
	forwarded := strings.Split(strings.Join(r.Header.Values("X-Forwarded-For"), ","), ",")
	for i := len(forwarded) - 1; i >= 0; i-- {
		address, err := netip.ParseAddr(strings.TrimSpace(forwarded[i]))
		if err != nil {
			return peer
		}

		address = address.Unmap()
		if !s.trustedProxy(address) {
			return address
		}
	}

	return peer
}

func (s *Server) beginPasswordCheck(w http.ResponseWriter) bool {
	select {
	case s.passwordChecks <- struct{}{}:
		return true
	default:
		w.Header().Set("Retry-After", "1")
		fail(w, http.StatusTooManyRequests, newError("rate_limited", "Too many requests."))
		return false
	}
}
