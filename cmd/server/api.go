// SPDX-FileCopyrightText: 2023 Institute for Automation of Complex Power Systems
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/VILLASframework/signaling/pkg"
	"github.com/gorilla/mux"
	"golang.org/x/exp/slog"
)

type apiErrorResponse struct {
	Error  string `json:"error"`
	Status string `json:"status"`
}

type apiSessionsResponse struct {
	Sessions []pkg.Session `json:"sessions"`
}

type apiSessionResponse struct {
	Session pkg.Session `json:"session"`
}

type apiPeerRequest struct {
	Peer *struct {
		Signals []pkg.Signal `json:"signals"`
	} `json:"peer"`
}

type apiPeerResponse struct {
	Peer pkg.Peer `json:"peer"`
}

func handleAPISessions(w http.ResponseWriter, r *http.Request) {
	resp := &apiSessionsResponse{}

	ss := []pkg.Session{}
	for _, s := range sessions {
		ss = append(ss, s.Marshal())
	}

	resp.Sessions = ss

	writeJSON(w, resp)
}

func handleAPISession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessName := vars["session"]

	// Create session if this is post request
	sess, ok := sessions[sessName]
	if !ok {
		if r.Method == "POST" {
			sess = NewSession(sessName)

			sessionsMutex.Lock()
			sessions[sessName] = sess
			sessionsMutex.Unlock()
		} else {
			writeError(w, http.StatusNotFound, fmt.Errorf("failed to find session with name '%s'", sessName))
			return
		}
	}

	resp := &apiSessionResponse{
		Session: sess.Marshal(),
	}

	writeJSON(w, resp)
}

func handleAPIPeer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessName := vars["session"]
	peerName := vars["peer"]

	// Create sess if this is post request
	var sess *Session
	if r.Method == "POST" {
		sess = GetSession(sessName)
	} else {
		sess = sessions[sessName]
		if sess == nil {
			writeError(w, http.StatusNotFound, fmt.Errorf("failed to find session with name '%s'", sessName))
			return
		}
	}

	peer, ok := sess.peers[peerName]
	if !ok {
		if r.Method == "POST" {
			var err error
			peer, err = sess.GetPeer(peerName)
			if err != nil {
				writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to create new peer: %w", err))
				return
			}
		} else {
			writeError(w, http.StatusNotFound, fmt.Errorf("failed to find peer with name '%s'", peerName))
			return
		}
	}

	if r.Method == "POST" {
		req := &apiPeerRequest{}
		if !readJSON(w, r, req) {
			return
		}

		if req.Peer == nil {
			writeError(w, http.StatusBadRequest, errors.New("malformed request body"))
			return
		}

		if sigs := req.Peer.Signals; sigs != nil {
			peer.logger.Debug("Updated signals", slog.Any("signals", sigs))

			peer.mutex.Lock()
			peer.signals = sigs
			peer.mutex.Unlock()
		}
	}

	resp := &apiPeerResponse{
		Peer: peer.Marshal(),
	}

	writeJSON(w, resp)
}
