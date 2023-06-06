package main

import "github.com/VILLASframework/signaling/pkg"

type SignalingMessage struct {
	pkg.SignalingMessage

	Sender *Connection
}

func (msg *SignalingMessage) CollectMetrics() {
	if msg.Candidate != nil {
		metricMessagesReceived.WithLabelValues("candidate").Inc()
	}
	if msg.Description != nil {
		metricMessagesReceived.WithLabelValues("description").Inc()
	}
}
