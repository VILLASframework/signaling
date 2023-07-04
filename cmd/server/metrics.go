// SPDX-FileCopyrightText: 2023 Institute for Automation of Complex Power Systems
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	_ = promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "signaling_active_sessions",
		Help: "The total number of active sessions",
	}, func() float64 {
		sessionsMutex.RLock()
		defer sessionsMutex.RUnlock()

		return float64(len(sessions))
	})

	_ = promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "signaling_active_peers",
		Help: "The total number of active connections",
	}, func() float64 {
		sessionsMutex.RLock()
		defer sessionsMutex.RUnlock()

		cnt := 0
		for _, s := range sessions {
			s.mutex.RLock()
			cnt += len(s.peers)
			s.mutex.RUnlock()
		}
		return float64(cnt)
	})

	metricSessionsCreated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "signaling_sessions",
		Help: "The total number of created sessions",
	})

	metricConnectionsCreated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "signaling_connections",
		Help: "The total number of created connections",
	})

	metricMessagesReceived = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "signaling_messages",
		Help: "The total number of messages exchanged",
	}, []string{"type"})

	metricHttpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Count of all HTTP requests",
	}, []string{"code", "method"})

	metricHttpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_request_duration_seconds",
		Help: "Duration of all HTTP requests",
	}, []string{"code", "method"})
)
