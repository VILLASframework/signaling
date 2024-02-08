// SPDX-FileCopyrightText: 2023 Institute for Automation of Complex Power Systems
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/VILLASframework/signaling/pkg"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	level  string

	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}

	sessions      = map[string]*Session{}
	sessionsMutex = sync.RWMutex{}
	server        *http.Server
)

func main() {
	flag.StringVar(&addr, "addr", ":8080", "http service address")
	flag.Var(&relays, "relay", "A TURN/STUN relay which is signalled to each connection (can be specified multiple times)")
	flag.StringVar(&level, "level", "debug", "The log level")
	flag.Parse()

	// h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
	// 	Level: slog.LevelDebug,
	// })
	// slog.SetDefault(slog.New(h))

	r := mux.NewRouter()

	a := r.PathPrefix("/api/v1").Subrouter()

	a.Use(
		func(next http.Handler) http.Handler {
			return promhttp.InstrumentHandlerCounter(metricHttpRequestsTotal, next)
		},
		func(next http.Handler) http.Handler {
			return promhttp.InstrumentHandlerDuration(metricHttpRequestDuration, next)
		},
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("Content-Type", "application/json")
				next.ServeHTTP(w, r)
			})
		},
	)

	a.Path("/sessions").
		Methods("GET").
		HandlerFunc(basicAuth(handleAPISessions))

	a.Path("/session/{session}").
		Methods("GET").
		HandlerFunc(handleAPISession)

	a.Path("/peer/{session}/{peer}").
		Methods("GET", "POST", "DELETE").
		HandlerFunc(handleAPIPeer)

	r.Path("/metrics").
		Methods("GET").
		Handler(promhttp.Handler())

	r.Path("/favicon.ico").
		Methods("GET").
		HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			http.Error(rw, "Not found", http.StatusNotFound)
		})

	r.Path("/healthz").
		Methods("GET", "OPTIONS").
		HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			rw.Write([]byte("OK")) //nolint:errcheck
		})

	r.Path("/{session}").
		HandlerFunc(handleWebsocket)

	r.Path("/{session}/{peer}").
		HandlerFunc(handleWebsocket)

	r.PathPrefix("/").
		HandlerFunc(handleWebsocket)	
		// HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 	slog.Info("Invalid request", slog.Any("path", r.URL.Path))
		// 	writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request"))
		// })

	expiryTicker := time.NewTicker(10 * time.Second)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	go func() {
		for {
			select {
			case <-expiryTicker.C:
				expireSessions()

			case sig := <-signals:
				slog.Debug("Received signal", slog.Any("signal", sig))

				closeSessions()

				if err := server.Shutdown(context.Background()); err != nil {
					slog.Error("Failed to shutdown HTTP server", slog.Any("error", err))
				}
			}
		}
	}()

	slog.Info("Listening", slog.String("addr", addr))

	server = &http.Server{
		Addr:    addr,
		Handler: r,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("Failed to listen and serve", slog.Any("error", err))
	}
}
