// SPDX-FileCopyrightText: 2023 Institute for Automation of Complex Power Systems
// SPDX-License-Identifier: Apache-2.0

package pkg

import (
	"encoding/json"
)

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

type RelayMessage struct {
	URL      string `json:"url"`
	Username string `json:"user"`
	Password string `json:"pass"`
	Realm    string `json:"realm"`
	Expires  string `json:"expires"`
}

type SignalingMessage struct {
	Relays      []RelayMessage      `json:"servers,omitempty"`
	Candidate   *CandidateMessage   `json:"candidate,omitempty"`
	Control     *ControlMessage     `json:"control,omitempty"`
	Description *DescriptionMessage `json:"description,omitempty"`
}

func (msg SignalingMessage) String() string {
	b, _ := json.Marshal(msg)
	return string(b)
}
