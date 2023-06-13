// SPDX-FileCopyrightText: 2023 Institute for Automation of Complex Power Systems
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/VILLASframework/signaling/pkg"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

type relayInfos []pkg.RelayInfo

func (r *relayInfos) String() string {
	strs := []string{}

	for _, r := range *r {
		strs = append(strs, r.URL)
	}

	return strings.Join(strs, ",")
}

func (r *relayInfos) Set(value string) error {
	ri, err := pkg.NewRelayInfo(value)
	if err != nil {
		return err
	}

	*r = append(*r, ri)

	return nil
}

var (
	// Flags
	addr   string
	relays relayInfos

	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}

	sessions      = map[string]*Session{}
	sessionsMutex = sync.Mutex{}
	server        *http.Server
)

func wsHandle(rw http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(rw, r, nil)
	if err != nil {
		logrus.Errorf("Failed to upgrade: %s", err)
		http.Error(rw, "Bad Request", http.StatusBadRequest)
		return
	}

	n := strings.TrimLeft(r.URL.Path, "/")
	if n == "" {
		logrus.Error("Empty session name")
		http.Error(rw, "Bad Request", http.StatusBadRequest)
		return
	}

	sessionsMutex.Lock()

	s, ok := sessions[n]
	if !ok {
		s = NewSession(n, relays)
		sessions[n] = s
	}

	if _, err := s.NewConnection(c, r); err != nil {
		logrus.Errorf("Failed to create connection: %s", err)
		http.Error(rw, "Internal Server Error", http.StatusInternalServerError)
	}

	sessionsMutex.Unlock()
}

func handleSignals(signals chan os.Signal) {
	for range signals {
		sessionsMutex.Lock()
		for _, s := range sessions {
			if err := s.Close(); err != nil {
				logrus.Panicf("Failed to close session: %s", err)
			}
		}
		sessionsMutex.Unlock()

		if err := server.Shutdown(context.Background()); err != nil {
			logrus.Panicf("Failed to shutdown HTTP server: %s", err)
		}
	}
}

func main() {
	flag.StringVar(&addr, "addr", ":8080", "http service address")
	flag.Var(&relays, "relay", "A TURN/STUN relay which is signalled to each connection (can be specified multiple times)")
	flag.Parse()

	signals := make(chan os.Signal, 10)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	// Block until signal is received
	go handleSignals(signals)

	server = &http.Server{
		Addr: addr,
	}

	handlerChain := promhttp.InstrumentHandlerDuration(metricHttpRequestDuration,
		promhttp.InstrumentHandlerCounter(metricHttpRequestsTotal,
			http.HandlerFunc(wsHandle),
		),
	)

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/favicon.ico", func(rw http.ResponseWriter, r *http.Request) {
		http.Error(rw, "Not found", http.StatusNotFound)
	})
	http.HandleFunc("/healthz", func(rw http.ResponseWriter, r *http.Request) {
		rw.Write([]byte("OK"))
	})
	http.HandleFunc("/api/v1/sessions", basicAuth(apiHandle))
	http.HandleFunc("/", handlerChain)

	logrus.Infof("Listening on: %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logrus.Errorf("Failed to listen and serve: %s", err)
	}
}
