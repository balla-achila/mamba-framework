package session

import (

    "context"
    "net/http"
    "time"
    "github.com/gorilla/sessions"
    "github.com/gorilla/securecookie"
)

type Session struct {
    session *sessions.Session
    request *http.Request
    writer  http.ResponseWriter
}

type Manager struct {
    store *sessions.CookieStore
    name  string
}

type Config struct {
    SecretKey string
    Name      string
    MaxAge    int
    Secure    bool
    HttpOnly  bool
    SameSite  string
}

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

func (m *Manager) Get(r *http.Request) (*Session, error) {
    session, err := m.store.Get(r, m.name)
    if err != nil {
        return nil, err
    }
    return &Session{
        session: session,
        request: r,
    }, nil
}

func (s *Session) Get(key string) interface{} {
    return s.session.Values[key]
}

func (s *Session) Set(key string, value interface{}) {
    s.session.Values[key] = value
}

func (s *Session) Delete(key string) {
    delete(s.session.Values, key)
}

func (s *Session) Clear() {
    s.session.Values = make(map[interface{}]interface{})
}

func (s *Session) Save() error {
    return s.session.Save(s.request, s.writer)
}

func (s *Session) IsAuthenticated() bool {
    userID, ok := s.Get("user_id").(int64)
    if !ok {
        return false
    }
    return userID > 0
}

func (s *Session) GetUserID() int64 {
    if userID, ok := s.Get("user_id").(int64); ok {
        return userID
    }
    return 0
}

func (s *Session) SetUser(userID int64, username, email string) {
    s.Set("user_id", userID)
    s.Set("username", username)
    s.Set("email", email)
    s.Set("authenticated", true)
    s.Set("last_activity", time.Now())
}

func (s *Session) Logout() {
    s.Clear()
    s.Set("authenticated", false)
}

func (s *Session) GetFlashMessages() map[string]string {
    if messages, ok := s.Get("flash_messages").(map[string]string); ok {
        s.Delete("flash_messages")
        return messages
    }
    return make(map[string]string)
}

func (s *Session) AddFlashMessage(key, message string) {
    messages := s.GetFlashMessages()
    messages[key] = message
    s.Set("flash_messages", messages)
}

func GenerateCSRF() string {
    return string(securecookie.GenerateRandomKey(32))
}
