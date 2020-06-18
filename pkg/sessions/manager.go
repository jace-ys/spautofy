package sessions

import (
	"encoding/gob"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/sessions"
)

func init() {
	gob.Register(sessionIDKey{})
}

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExists   = errors.New("existing session found")
)

type sessionIDKey struct{}

type Session struct {
	*sessions.Session
}

func (s *Session) GetID() string {
	id, ok := s.Values[sessionIDKey{}].(string)
	if !ok {
		return ""
	}
	return id
}

type Manager struct {
	store sessions.Store
	name  string
}

func NewManager(name, key string, duration time.Duration) *Manager {
	store := sessions.NewCookieStore([]byte(key))
	store.Options = &sessions.Options{
		Path:     "/",
		HttpOnly: true,
		MaxAge:   int(duration / time.Second),
	}

	return &Manager{
		store: store,
		name:  name,
	}
}

func (m *Manager) Get(r *http.Request) (*Session, error) {
	session, err := m.store.Get(r, m.name)
	if err != nil {
		return nil, err
	}

	return &Session{session}, nil
}

func (m *Manager) Create(w http.ResponseWriter, r *http.Request) (*Session, error) {
	session, err := m.store.Get(r, m.name)
	if err != nil {
		return nil, err
	}

	session.Values[sessionIDKey{}] = uuid.New().String()

	err = session.Save(r, w)
	if err != nil {
		return nil, err
	}

	return &Session{session}, nil
}

func (m *Manager) CreateWithValues(w http.ResponseWriter, r *http.Request, values map[interface{}]interface{}) (*Session, error) {
	session, err := m.store.Get(r, m.name)
	if err != nil {
		return nil, err
	}

	for k, v := range values {
		session.Values[k] = v
	}

	session.Values[sessionIDKey{}] = uuid.New().String()

	err = session.Save(r, w)
	if err != nil {
		return nil, err
	}

	return &Session{session}, nil
}

func (m *Manager) Update(w http.ResponseWriter, r *http.Request, values map[interface{}]interface{}) (*Session, error) {
	session, err := m.store.Get(r, m.name)
	if err != nil {
		return nil, err
	}

	for k, v := range values {
		session.Values[k] = v
	}

	err = session.Save(r, w)
	if err != nil {
		return nil, err
	}

	return &Session{session}, nil
}

func (m *Manager) Delete(w http.ResponseWriter, r *http.Request) (*Session, error) {
	session, err := m.store.Get(r, m.name)
	if err != nil {
		return nil, err
	}

	session.Options.MaxAge = -1

	err = session.Save(r, w)
	if err != nil {
		return nil, err
	}

	return &Session{session}, nil
}
