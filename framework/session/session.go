package session

import (
	"net/http"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

// Session wraps a gorilla session
type Session struct {
	session *sessions.Session
	request *http.Request
	writer  http.ResponseWriter
}

// Manager manages sessions
type Manager struct {
	store *sessions.CookieStore
	name  string
}

// Config holds session configuration
type Config struct {
	SecretKey string
	Name      string
	MaxAge    int
	Secure    bool
	HttpOnly  bool
	SameSite  string
}

// New creates a new session manager
func New(cfg *Config) *Manager {
	key := []byte(cfg.SecretKey)
	store := sessions.NewCookieStore(key)

	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   cfg.MaxAge,
		Secure:   cfg.Secure,
		HttpOnly: cfg.HttpOnly,
		SameSite: parseSameSite(cfg.SameSite),
	}

	return &Manager{
		store: store,
		name:  cfg.Name,
	}
}

// parseSameSite converts string to http.SameSite
func parseSameSite(s string) http.SameSite {
	switch s {
	case "lax":
		return http.SameSiteLaxMode
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteDefaultMode
	}
}

// Middleware adds session to request context
func (m *Manager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := m.store.Get(r, m.name)
		ctx := ContextWithSession(r.Context(), &Session{
			session: session,
			request: r,
			writer:  w,
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Get retrieves a session from the request. w is required so the returned
// Session can later be persisted via Save() (which needs a ResponseWriter
// to set the session cookie) -- without it, Save() would nil-pointer panic.
func (m *Manager) Get(w http.ResponseWriter, r *http.Request) (*Session, error) {
	session, err := m.store.Get(r, m.name)
	if err != nil {
		return nil, err
	}
	return &Session{
		session: session,
		request: r,
		writer:  w,
	}, nil
}

// Get returns a value from the session
func (s *Session) Get(key string) interface{} {
	return s.session.Values[key]
}

// Set stores a value in the session
func (s *Session) Set(key string, value interface{}) {
	s.session.Values[key] = value
}

// Delete removes a value from the session
func (s *Session) Delete(key string) {
	delete(s.session.Values, key)
}

// Clear removes all values from the session
func (s *Session) Clear() {
	s.session.Values = make(map[interface{}]interface{})
}

// Save saves the session
func (s *Session) Save() error {
	return s.session.Save(s.request, s.writer)
}

// IsAuthenticated checks if the user is authenticated
func (s *Session) IsAuthenticated() bool {
	userID, ok := s.Get("user_id").(int64)
	if !ok {
		return false
	}
	return userID > 0
}

// GetUserID returns the user ID from the session
func (s *Session) GetUserID() int64 {
	if userID, ok := s.Get("user_id").(int64); ok {
		return userID
	}
	return 0
}

// SetUser sets the user in the session
func (s *Session) SetUser(userID int64, username, email string) {
	s.Set("user_id", userID)
	s.Set("username", username)
	s.Set("email", email)
	s.Set("authenticated", true)
	s.Set("last_activity", time.Now())
}

// Logout clears the session
func (s *Session) Logout() {
	s.Clear()
	s.Set("authenticated", false)
}

// GetFlashMessages retrieves and clears flash messages
func (s *Session) GetFlashMessages() map[string]string {
	if messages, ok := s.Get("flash_messages").(map[string]string); ok {
		s.Delete("flash_messages")
		return messages
	}
	return make(map[string]string)
}

// AddFlashMessage adds a flash message
func (s *Session) AddFlashMessage(key, message string) {
	messages := s.GetFlashMessages()
	messages[key] = message
	s.Set("flash_messages", messages)
}

// GenerateCSRF generates a CSRF token
func GenerateCSRF() string {
	return string(securecookie.GenerateRandomKey(32))
}
