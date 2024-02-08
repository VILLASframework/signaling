// SPDX-FileCopyrightText: 2023 Institute for Automation of Complex Power Systems
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

func writeJSON(w http.ResponseWriter, resp any) bool {
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("Failed to encode API response",
			slog.Any("error", err),
			slog.Any("resp", resp))
		w.WriteHeader(http.StatusInternalServerError)

		return false
	}

	return true
}

func readJSON(w http.ResponseWriter, r *http.Request, req any) bool {
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("failed to parse request body: %w", err))
		return false
	}

	return true
}

func writeError(w http.ResponseWriter, code int, err error) bool {
	resp := &apiErrorResponse{
		Error:  err.Error(),
		Status: http.StatusText(code),
	}

	slog.Error("Request failed", slog.Any("error", err))

	if writeJSON(w, resp) {
		w.WriteHeader(code)

		return true
	}

	return false
}
