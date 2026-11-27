package main

import (
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/idristarwala/agent-factory/internal/artifact"
	"github.com/idristarwala/agent-factory/internal/revision"
)

func chatArtifactRoute(mux *http.ServeMux, s revision.Store) {
	mux.HandleFunc("/api/chat/github", apiRoute(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "use POST", 405)
			return
		}
		var req struct {
			artifact.Ref
			URL string
		}
		if !readEdit(w, r, &req) {
			return
		}
		var github artifact.GitHub
		if os.Getenv("API_TOKEN") != "" {
			github.Token = os.Getenv("FACTORY_GITHUB_TOKEN")
			github.Repositories = strings.Split(os.Getenv("FACTORY_GITHUB_REPOSITORIES"), ",")
		}
		file, err := github.Import(r.Context(), s, req.Ref, req.URL)
		if err != nil && !errors.Is(err, revision.ErrConflict) {
			http.Error(w, err.Error(), 422)
			return
		}
		sendJSON(w, file, err)
	}, "POST"))
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
