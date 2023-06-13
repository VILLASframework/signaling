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

type Connection struct {
	pkg.Connection

	*websocket.Conn

	Session *Session

	Messages chan SignalingMessage

	close chan struct{}
	done  chan struct{}

	Closing bool
}

func (s *Session) NewConnection(c *websocket.Conn, r *http.Request) (*Connection, error) {
	slog.Info("New connection",
		slog.Any("remote", c.RemoteAddr()),
		slog.Any("session", s))

	d := &Connection{
		Connection: pkg.Connection{
			Created:   time.Now(),
			Remote:    r.RemoteAddr,
			UserAgent: r.UserAgent(),
		},
		Conn:     c,
		Session:  s,
		Messages: make(chan SignalingMessage),
		close:    make(chan struct{}),
		done:     make(chan struct{}),
	}

	if err := s.AddConnection(d); err != nil {
		return nil, fmt.Errorf("failed to add connection: %w", err)
	}

	d.Conn.SetReadLimit(maxMessageSize)
	if err := d.Conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}
	d.Conn.SetPongHandler(func(string) error {
		return d.Conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	go d.read()
	go d.run()

	metricConnectionsCreated.Inc()

	return d, nil
}

func (d *Connection) String() string {
	return d.Conn.RemoteAddr().String()
}

func (d *Connection) read() {
	for {
		var msg pkg.SignalingMessage
		if err := d.Conn.ReadJSON(&msg); err != nil {
			if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				if !d.Closing {
					d.Closing = true
					err := d.Conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(5*time.Second))
					if err != nil && err != websocket.ErrCloseSent {
						slog.Error("Failed to send close message", slog.Any("error", err))
					}
				}
			} else {
				slog.Error("Failed to read", slog.Any("error", err))
			}
			break
		}

		slog.Info("Read signaling message",
			slog.Any("remote", d.Conn.RemoteAddr()),
			slog.Any("msg", msg))
		d.Session.Messages <- SignalingMessage{
			SignalingMessage: msg,
			Sender:           d,
		}
	}

	d.closed()
}

func (d *Connection) run() {
	ticker := time.NewTicker(pingPeriod)

loop:
	for {
		select {

		case <-d.done:
			break loop

		case msg, ok := <-d.Messages:
			if !ok {
				d.Close()
				break loop
			}

			slog.Info("Sending",
				slog.Any("from", msg.Sender),
				slog.Any("to", d.Conn.RemoteAddr()),
				slog.Any("msg", msg))

			if err := d.Conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				slog.Error("Failed to set read deadline", slog.Any("error", err))
			}
			if err := d.Conn.WriteJSON(msg.SignalingMessage); err != nil {
				slog.Error("Failed to send message", slog.Any("error", err))
			}

		case <-ticker.C:
			slog.Debug("Send ping message")

			if err := d.Conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				slog.Error("Failed to set write deadline", slog.Any("error", err))
			}
			if err := d.Conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second)); err != nil {
				slog.Error("Failed to ping", slog.Any("error", err))
			}
		}
	}
}

func (d *Connection) Close() error {
	if d.Closing {
		return errors.New("connection is closing")
	}

	d.Closing = true
	slog.Info("Connection closing", slog.Any("remote", d.Conn.RemoteAddr()))

	if err := d.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
		return fmt.Errorf("failed to send close message: %w", err)
	}

	select {
	case <-d.done:
	case <-time.After(time.Second):
		slog.Warn("Timed-out waiting for connection close")
	}

	return nil
}

func (d *Connection) closed() {
	close(d.done)

	if err := d.Conn.Close(); err != nil {
		slog.Error("Failed to close connection", slog.Any("error", err))
	}

	slog.Info("Connection closed", slog.Any("conn", d))

	if err := d.Session.RemoveConnection(d); err != nil {
		slog.Warn("Failed to remove connection", slog.Any("conn", d))
	}
}
