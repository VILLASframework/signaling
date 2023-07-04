// SPDX-FileCopyrightText: 2023 Institute for Automation of Complex Power Systems
// SPDX-License-Identifier: Apache-2.0

package main

import "github.com/VILLASframework/signaling/pkg"

type SignalingMessage struct {
	pkg.SignalingMessage

	Sender *Peer
}

func (msg *SignalingMessage) CollectMetrics() {
	if msg.Candidate != nil {
		metricMessagesReceived.WithLabelValues("candidate").Inc()
	}
	if msg.Description != nil {
		metricMessagesReceived.WithLabelValues("description").Inc()
	}
	if msg.Control != nil {
		metricMessagesReceived.WithLabelValues("control").Inc()
	}
	if msg.Signals != nil {
		metricMessagesReceived.WithLabelValues("signals").Inc()
	}
}
