// SPDX-FileCopyrightText: 2023 Institute for Automation of Complex Power Systems
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"sync"
	"time"

	"github.com/VILLASframework/signaling/pkg"
	"golang.org/x/exp/slog"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 10 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 4096
)

type Peer struct {
	Name      string
	created   time.Time
	id        int32
	signals   []pkg.Signal
	userAgent string
	connected time.Time

	conn    *Connection
	session *Session

	mutex sync.RWMutex

	logger *slog.Logger
}

func (s *Session) NewPeer(name string) (*Peer, error) {
	d := &Peer{
		Name:    name,
		created: time.Now(),
		session: s,
		logger:  s.logger.With(slog.String("peer", name)),
	}

	d.logger.Info("New peer")

	metricConnectionsCreated.Inc()

	return d, nil
}

func (p *Peer) String() string {
	return p.conn.RemoteAddr().String()
}

func (p *Peer) Marshal() pkg.Peer {
	pm := pkg.Peer{
		Name:      p.Name,
		ID:        p.id,
		UserAgent: p.userAgent,
		Created:   p.created,
		Signals:   p.signals,
	}

	if p.conn != nil {
		pm.Remote = p.conn.RemoteAddr().String()
		pm.Connected = p.connected
	}

	return pm
}

func (p *Peer) Close() error {
	if p.conn != nil {
		if err := p.conn.Close(); err != nil {
			return err
		}
	}

	return nil
}
