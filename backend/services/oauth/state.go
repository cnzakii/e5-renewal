package oauth

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// OAuthState stores the account information when an OAuth flow is initiated.
type OAuthState struct {
	ClientID     string
	ClientSecret string
	TenantID     string
	RedirectURI  string
	CreatedAt    time.Time
	TTL          time.Duration
}

const defaultStateTTL = 5 * time.Minute

type StateStore struct {
	mu     sync.Mutex
	states map[string]OAuthState
}

var GlobalStateStore = &StateStore{
	states: make(map[string]OAuthState),
}

// NewState generates and stores a state token that automatically expires after the configured TTL.
// If TTL is zero, it defaults to 5 minutes.
func (s *StateStore) NewState(data OAuthState) string {
	id := uuid.NewString()
	data.CreatedAt = time.Now()
	if data.TTL == 0 {
		data.TTL = defaultStateTTL
	}
	s.mu.Lock()
	s.states[id] = data
	s.mu.Unlock()
	// Clean up after TTL
	time.AfterFunc(data.TTL, func() {
		s.mu.Lock()
		delete(s.states, id)
		s.mu.Unlock()
	})
	return id
}

// Reset clears all states — for testing only
func (s *StateStore) Reset() {
	s.mu.Lock()
	s.states = make(map[string]OAuthState)
	s.mu.Unlock()
}

// Consume retrieves and deletes the state (single-use, to prevent replay attacks).
func (s *StateStore) Consume(id string) (OAuthState, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.states[id]
	if !ok {
		return OAuthState{}, false
	}
	delete(s.states, id)
	ttl := data.TTL
	if ttl == 0 {
		ttl = defaultStateTTL
	}
	if time.Since(data.CreatedAt) > ttl {
		return OAuthState{}, false
	}
	return data, true
}
