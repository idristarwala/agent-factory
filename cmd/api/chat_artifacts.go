package main

import (
	"errors"
	"github.com/idristarwala/agent-factory/internal/secrets"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/idristarwala/agent-factory/internal/artifact"
	"github.com/idristarwala/agent-factory/internal/revision"
)

func chatArtifactRoute(mux *http.ServeMux, s revision.Store) {
	secretLoader := secrets.New()
	mux.HandleFunc("/api/chat/artifacts/{id}/{version}", apiRoute(func(w http.ResponseWriter, r *http.Request) {
		version, err := strconv.Atoi(r.PathValue("version"))
		if err != nil {
			http.Error(w, "invalid artifact revision", 400)
			return
		}
		f, err := artifact.ReadFile(s, artifact.Ref{ID: r.PathValue("id"), Version: version})
		if err != nil {
			http.Error(w, err.Error(), 404)
			return
		}
		// Return only the requested location rather than all original document bytes.
		loc := r.URL.Query().Get("location")
		switch {
		case loc == "file" && f.MIME == "text/plain":
			sendJSON(w, map[string]string{"text": string(f.Data)}, nil)
		case loc == "image" && strings.HasPrefix(f.MIME, "image/"):
			sendJSON(w, map[string]any{"mime": f.MIME, "image": f.Data}, nil)
		default:
			for _, p := range f.Passages {
				if p.Location == loc {
					sendJSON(w, map[string]string{"text": p.Text}, nil)
					return
				}
			}
			http.Error(w, "unknown citation location", 404)
		}
	}, "GET"))
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
			github.Token, _ = secretLoader.Get("factory-github-token")
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
