// SPDX-FileCopyrightText: 2023 Institute for Automation of Complex Power Systems
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/VILLASframework/signaling/pkg"
	"golang.org/x/exp/slog"
)

type Session struct {
	Name    string
	Created time.Time

	Messages chan SignalingMessage

	Relays           []pkg.RelayInfo
	Connections      map[*Connection]interface{}
	ConnectionsMutex sync.RWMutex

	LastConnectionID int
}

func NewSession(name string, relays []pkg.RelayInfo) *Session {
	slog.Info("Session opened", slog.String("name", name))

	s := &Session{
		Name:             name,
		Created:          time.Now(),
		Relays:           relays,
		Connections:      map[*Connection]interface{}{},
		Messages:         make(chan SignalingMessage, 100),
		LastConnectionID: 0,
	}

	go s.run()

	metricSessionsCreated.Inc()

	return s
}

func (s *Session) RemoveConnection(c *Connection) error {
	s.ConnectionsMutex.Lock()
	defer s.ConnectionsMutex.Unlock()

	delete(s.Connections, c)

	if len(s.Connections) == 0 {
		sessionsMutex.Lock()
		delete(sessions, s.Name)
		sessionsMutex.Unlock()

		slog.Info("Session closed", slog.String("name", s.Name))

		return nil
	} else {
		return s.SendControlMessages()
	}
}

func (s *Session) AddConnection(c *Connection) error {
	s.ConnectionsMutex.Lock()
	defer s.ConnectionsMutex.Unlock()

	c.ID = s.LastConnectionID
	s.LastConnectionID++

	s.Connections[c] = nil

	if err := s.SendRelaysMessage(c); err != nil {
		return fmt.Errorf("failed to send relays message: %w", err)
	}

	if err := s.SendControlMessages(); err != nil {
		return fmt.Errorf("failed to send control messages: %w", err)
	}

	return nil
}

func (s *Session) SendRelaysMessage(c *Connection) error {
	cmsg := &pkg.SignalingMessage{}
	for _, relay := range s.Relays {
		user, pass, exp := relay.GetCredentials("villas")
		cmsg.Relays = append(cmsg.Relays, pkg.RelayMessage{
			URL:      relay.URL,
			Username: user,
			Password: pass,
			Realm:    relay.Realm,
			Expires:  exp.Format(time.RFC3339),
		})
	}

	return c.Conn.WriteJSON(cmsg)
}

func (s *Session) SendControlMessages() error {
	conns := []pkg.Connection{}
	for c := range s.Connections {
		conns = append(conns, c.Connection)
	}

	cmsg := &pkg.SignalingMessage{
		Control: &pkg.ControlMessage{
			Connections: conns,
		},
	}

	for c := range s.Connections {
		cmsg.Control.ConnectionID = c.ID

		if err := c.Conn.WriteJSON(cmsg); err != nil {
			return err
		} else {
			slog.Info("Send control message", slog.Any("msg", cmsg))
		}
	}

	return nil
}

func (s *Session) String() string {
	return s.Name
}

func (s *Session) run() {
	for msg := range s.Messages {
		msg.CollectMetrics()

		s.ConnectionsMutex.RLock()

		for c := range s.Connections {
			if msg.Sender != c {
				c.Messages <- msg
			}
		}

		s.ConnectionsMutex.RUnlock()
	}
}

func (s *Session) Close() error {
	s.ConnectionsMutex.Lock()
	defer s.ConnectionsMutex.Unlock()

	for c := range s.Connections {
		if err := c.Close(); err != nil {
			return err
		}
	}

	return nil
}
