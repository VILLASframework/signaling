// SPDX-FileCopyrightText: 2023 Institute for Automation of Complex Power Systems
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/VILLASframework/signaling/pkg"
)

const sessionExpiryAge = time.Hour

type Session struct {
	Name    string
	Created time.Time

	messages chan SignalingMessage

	peers map[string]*Peer
	mutex sync.RWMutex

	lastPeerID atomic.Int32

	logger *slog.Logger
}

func NewSession(name string) *Session {
	s := &Session{
		Name:     name,
		Created:  time.Now(),
		peers:    map[string]*Peer{},
		messages: make(chan SignalingMessage, 100),

		logger: slog.With(slog.String("session", name)),
	}

	s.logger.Info("Session opened")

	go s.run()

	metricSessionsCreated.Inc()

	return s
}

func GetSession(name string) *Session {
	sessionsMutex.RLock()
	defer sessionsMutex.RUnlock()

	return sessions[name]
}

func GetOrCreateSession(name string) (*Session, error) {
	sessionsMutex.Lock()
	defer sessionsMutex.Unlock()

	s := sessions[name]
	if s == nil {
		s = NewSession(name)
		sessions[name] = s
	}

	return s, nil
}

func (s *Session) RemovePeer(p *Peer) error {
	if err := p.Close(); err != nil {
		return fmt.Errorf("failed to close peer: %w", err)
	}

	s.mutex.Lock()
	delete(s.peers, p.Name)
	s.mutex.Unlock()

	return nil
}

func (s *Session) SendControlMessageToAllConnectedPeers() error {
	msg := &pkg.SignalingMessage{
		Control: &pkg.ControlMessage{},
	}

	s.mutex.RLock()
	for _, p := range s.peers {
		msg.Control.Peers = append(msg.Control.Peers, p.Marshal())
	}
	s.mutex.RUnlock()

	for _, p := range s.peers {
		msg.Control.PeerID = p.id

		if p.conn == nil {
			continue
		}

		if err := p.conn.WriteJSON(msg); err != nil {
			return err
		} else {
			s.logger.Info("Send control message", slog.Any("msg", msg))
		}
	}

	return nil
}

func (s *Session) String() string {
	return s.Name
}

func (s *Session) Close() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, p := range s.peers {
		if err := p.Close(); err != nil {
			return err
		}
	}

	return nil
}

func (s *Session) run() {
	for msg := range s.messages {
		s.handleMessage(msg)
	}
}

func (s *Session) handleMessage(msg SignalingMessage) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	msg.CollectMetrics()

	for _, p := range s.peers {
		if msg.Sender == p || p.conn == nil {
			continue
		}

		p.conn.messages <- msg
	}
}

func (s *Session) Marshal() pkg.Session {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	conns := []pkg.Peer{}
	for _, p := range s.peers {
		conns = append(conns, p.Marshal())
	}

	return pkg.Session{
		Name:    s.Name,
		Created: s.Created,
		Peers:   conns,
	}
}

func (s *Session) GetPeer(name string) (p *Peer) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.peers[name]
}

func (s *Session) GetOrCreatePeer(name string) (p *Peer, err error) {
	var ok bool

	s.mutex.Lock()
	defer s.mutex.Unlock()

	p, ok = s.peers[name]
	if !ok {
		p, err = s.NewPeer(name)
		if err != nil {
			return nil, err
		}

		s.peers[p.Name] = p
	}

	return p, nil
}

func closeSessions() {
	sessionsMutex.Lock()
	defer sessionsMutex.Unlock()

	for name, s := range sessions {
		if err := s.Close(); err != nil {
			slog.Error("Failed to close session", slog.Any("error", err))
		}

		delete(sessions, name)
	}
}

func expireSessions() {
	sessionsMutex.Lock()
	defer sessionsMutex.Unlock()

	for name, session := range sessions {
		if len(session.peers) == 0 && time.Since(session.Created) > sessionExpiryAge {
			slog.Debug("Removing stale session",
				slog.String("session", name),
				slog.Time("created", session.Created))

			if err := session.Close(); err != nil {
				slog.Error("Failed to close session", slog.Any("error", err))
			}

			delete(sessions, name)
		}
	}
}
