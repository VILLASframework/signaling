// SPDX-FileCopyrightText: 2023 Institute for Automation of Complex Power Systems
// SPDX-License-Identifier: Apache-2.0

package pkg

import "time"

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
