package main

import (
	"errors"
	"net/http"

	"github.com/idristarwala/agent-factory/internal/artifact"
	"github.com/idristarwala/agent-factory/internal/revision"
)

func chatArtifactRoute(mux *http.ServeMux, s revision.Store) {
	mux.HandleFunc("/api/chat/artifacts", apiRoute(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "use POST", 405)
			return
		}
		var upload struct {
			artifact.Ref
			Name string
			Data []byte
		}
		if !readEdit(w, r, &upload) {
			return
		}
		ref, err := artifact.Save(s, upload.Ref, upload.Name, upload.Data)
		if err != nil && !errors.Is(err, revision.ErrConflict) {
			http.Error(w, err.Error(), 422)
			return
		}
		sendJSON(w, ref, err)
	}, "POST"))
}
