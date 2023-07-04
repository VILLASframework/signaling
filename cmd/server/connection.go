// SPDX-FileCopyrightText: 2023 Institute for Automation of Complex Power Systems
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/VILLASframework/signaling/pkg"
	"github.com/gorilla/websocket"
	"golang.org/x/exp/slog"
)

type Connection struct {
	*websocket.Conn

	peer *Peer

	messages chan SignalingMessage
	closing  bool
	close    chan struct{}
	done     chan struct{}

	logger *slog.Logger
}

func (p *Peer) Connect(w http.ResponseWriter, r *http.Request) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.conn != nil {
		return errors.New("peer is already connected")
	}

	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return fmt.Errorf("failed to upgrade connection: %w", err)
	}

	p.id = p.session.lastPeerID.Add(1)
	p.userAgent = r.UserAgent()
	p.connected = time.Now()
	p.conn = &Connection{
		Conn:     wsConn,
		peer:     p,
		messages: make(chan SignalingMessage),
		close:    make(chan struct{}),
		done:     make(chan struct{}),
		logger:   p.logger.With(slog.String("remote", r.RemoteAddr)),
	}

	p.conn.SetReadLimit(maxMessageSize)
	if err := p.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		return fmt.Errorf("failed to set read deadline: %w", err)
	}

	p.conn.SetPongHandler(func(string) error {
		return p.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	if err := p.conn.RecvSignalsMessage(); err != nil {
		return fmt.Errorf("failed to receive signals message: %w", err)
	}

	if err := p.conn.SendRelaysMessage(); err != nil {
		return fmt.Errorf("failed to send relays message: %w", err)
	}

	if err := p.session.SendControlMessageToAllConnectedPeers(); err != nil {
		return fmt.Errorf("failed to send control messages: %w", err)
	}

	go p.conn.read()
	go p.conn.run()

	return nil
}

func (c *Connection) Close() error {
	if c.closing {
		return errors.New("connection is closing")
	}

	c.closing = true
	c.logger.Info("Connection closing")

	if err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
		return fmt.Errorf("failed to send close message: %w", err)
	}

	select {
	case <-c.done:
	case <-time.After(time.Second):
		c.logger.Warn("Timed-out waiting for connection close")
	}

	c.peer.connected = time.Time{}
	c.peer.conn = nil

	return nil
}

func (c *Connection) RecvSignalsMessage() error {
	msg := &pkg.SignalingMessage{}

	if err := c.ReadJSON(msg); err != nil {
		return fmt.Errorf("failed to read signaling message: %w", err)
	}

	// TODO: Wait until we get valid signals from node
	if false && msg.Signals != nil {
		c.peer.mutex.Lock()
		c.peer.signals = msg.Signals
		c.peer.mutex.Unlock()

		c.logger.Debug("Received signals", slog.Any("signals", msg.Signals))
	}

	return nil
}

func (c *Connection) SendRelaysMessage() error {
	msg := &pkg.SignalingMessage{}

	for _, relay := range relays {
		user, pass, exp := relay.GetCredentials("villas")
		msg.Relays = append(msg.Relays, pkg.Relay{
			URL:      relay.URL,
			Username: user,
			Password: pass,
			Realm:    relay.Realm,
			Expires:  exp.Format(time.RFC3339),
		})
	}

	return c.WriteJSON(msg)
}

func (c *Connection) handleMessage(msg pkg.SignalingMessage) {
	c.logger.Info("Received signaling message", slog.Any("msg", msg))

	c.peer.session.messages <- SignalingMessage{
		SignalingMessage: msg,
		Sender:           c.peer,
	}
}

func (c *Connection) read() {
	for {
		msg := pkg.SignalingMessage{}
		if err := c.ReadJSON(&msg); err != nil {
			if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				if !c.closing {
					c.closing = true
					err := c.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(5*time.Second))
					if err != nil && err != websocket.ErrCloseSent {
						c.logger.Error("Failed to send close message", slog.Any("error", err))
					}
				}
			} else {
				c.logger.Error("Failed to read", slog.Any("error", err))
			}
			break
		}

		c.handleMessage(msg)
	}

	c.closed()
}

func (c *Connection) run() {
	ticker := time.NewTicker(pingPeriod)

loop:
	for {
		select {

		case <-c.done:
			break loop

		case msg, ok := <-c.messages:
			if !ok {
				c.Close()
				break loop
			}

			c.logger.Info("Sending signaling message",
				slog.Any("from", msg.Sender),
				slog.Any("msg", msg))

			if err := c.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				c.logger.Error("Failed to set read deadline", slog.Any("error", err))
			}

			if err := c.WriteJSON(msg.SignalingMessage); err != nil {
				c.logger.Error("Failed to send message", slog.Any("error", err))
			}

		case <-ticker.C:
			c.logger.Debug("Send ping message")

			if err := c.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				c.logger.Error("Failed to set write deadline", slog.Any("error", err))
			}

			if err := c.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second)); err != nil {
				c.logger.Error("Failed to ping", slog.Any("error", err))
			}
		}
	}
}

func (c *Connection) closed() {
	close(c.done)

	if err := c.Conn.Close(); err != nil {
		c.logger.Error("Failed to close connection", slog.Any("error", err))
	}

	c.logger.Info("Connection closed")

	c.peer.conn = nil
}
