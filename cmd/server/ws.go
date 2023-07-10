// SPDX-FileCopyrightText: 2023 Institute for Automation of Complex Power Systems
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func handleWebsocket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	sessName := vars["session"]
	peerName, ok := vars["peer"]
	if !ok {
		peerName = uuid.New().String()
	}

	sess, err := GetOrCreateSession(sessName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to create session: %w", err))
		return
	}

	peer, err := sess.GetOrCreatePeer(peerName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to create peer: %w", err))
		return
	}

	if err := peer.Connect(w, r); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("failed to connect: %w", err))
		return
	}
}
