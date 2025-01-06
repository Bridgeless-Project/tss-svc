package session

import (
	"sync"

	"github.com/hyle-team/tss-svc/internal/p2p"
	"github.com/pkg/errors"
)

var (
	ErrSessionNotFound = errors.New("session not found")
)

type Session interface {
	Id() string
	Receive(request *p2p.SubmitRequest) error
	RegisterIdChangeListener(func(oldId, newId string))
}

type Manager struct {
	sessions map[string]Session
	mu       sync.RWMutex
}

func (m *Manager) AddSession(session Session) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sessions[session.Id()] = session
	session.RegisterIdChangeListener(m.onIdChange)
}

func (m *Manager) Get(id string) Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if session, exists := m.sessions[id]; exists {
		return session
	}

	return nil
}

func (m *Manager) Receive(request *p2p.SubmitRequest) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if session, exists := m.sessions[request.SessionId]; exists {
		return session.Receive(request)
	}

	return ErrSessionNotFound
}

func (m *Manager) onIdChange(oldId, newId string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[oldId]
	if !ok {
		return
	}

	delete(m.sessions, oldId)
	m.sessions[newId] = session
	// id change listener remains the same
}
