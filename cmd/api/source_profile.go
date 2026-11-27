package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/idristarwala/agent-factory/internal/mapping"
)

func profileSource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			http.Error(w, "sample exceeds 1 MiB", http.StatusRequestEntityTooLarge)
		} else {
			http.Error(w, "unreadable sample", http.StatusBadRequest)
		}
		return
	}
	if !json.Valid(data) {
		http.Error(w, "supply one JSON sample", http.StatusBadRequest)
		return
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var sample any
	if err := dec.Decode(&sample); err != nil {
		http.Error(w, "invalid JSON sample", http.StatusBadRequest)
		return
	}
	profiles, err := mapping.ProfileSample(sample)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"collections": profiles})
}
