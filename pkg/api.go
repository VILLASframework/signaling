// SPDX-FileCopyrightText: 2023 Institute for Automation of Complex Power Systems
// SPDX-License-Identifier: Apache-2.0

package pkg

import "time"

type Session struct {
	Name    string    `json:"name"`
	Created time.Time `json:"created"`
	Peers   []Peer    `json:"peers"`
}

type SignalType string

const (
	SignalTypeFloat   SignalType = "float"
	SignalTypeInteger SignalType = "integer"
	SignalTypeBoolean SignalType = "boolean"
	SignalTypeComplex SignalType = "complex"
)

type Signal struct {
	Name string     `json:"name"`
	Type SignalType `json:"type"`
	Unit string     `json:"unit,omitempty"`
	Init any        `json:"init,omitempty"`
}

type Peer struct {
	Name      string    `json:"name"`
	ID        int32     `json:"id,omitempty"`
	Remote    string    `json:"remote,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
	Created   time.Time `json:"created"`
	Connected time.Time `json:"connected,omitempty"`
	Signals   []Signal  `json:"signals,omitempty"`
}
