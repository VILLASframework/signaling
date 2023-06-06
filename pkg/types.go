// SPDX-FileCopyrightText: 2023 Institute for Automation of Complex Power Systems
// SPDX-License-Identifier: Apache-2.0

package pkg

import (
	"encoding/json"
	"time"
)

type Session struct {
	Name        string       `json:"name"`
	Created     time.Time    `json:"created"`
	Connections []Connection `json:"connections"`
}

type Connection struct {
	ID        int       `json:"id"`
	Remote    string    `json:"remote"`
	UserAgent string    `json:"user_agent"`
	Created   time.Time `json:"created"`
}

type ControlMessage struct {
	ConnectionID int          `json:"connection_id"`
	Connections  []Connection `json:"connections"`
}

type DescriptionMessage struct {
	Spd  string `json:"spd"`
	Type string `json:"type"`
}

type CandidateMessage struct {
	Spd string `json:"spd"`
	Mid string `json:"mid"`
}

type SignalingMessage struct {
	Candidate   *CandidateMessage   `json:"candidate,omitempty"`
	Control     *ControlMessage     `json:"control,omitempty"`
	Description *DescriptionMessage `json:"description,omitempty"`
}

func (msg SignalingMessage) String() string {
	b, _ := json.Marshal(msg)
	return string(b)
}
