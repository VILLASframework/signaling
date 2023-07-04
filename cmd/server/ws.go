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
	connName, ok := vars["connection"]
	if !ok {
		connName = uuid.New().String()
	}

	sess := GetSession(sessName)
	peer, err := sess.GetPeer(connName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to create connection: %w", err))
		return
	}

	if err := peer.Connect(w, r); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("failed to connect: %w", err))
		return
	}
}
