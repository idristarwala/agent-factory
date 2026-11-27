package main

import (
	"errors"
	"io/fs"
	"net/http"
	"strconv"
	"strings"

	"github.com/idristarwala/agent-factory/internal/memory"
	"github.com/idristarwala/agent-factory/internal/revision"
)

func memoryRoutes(mux *http.ServeMux, store revision.Store) {
	mux.HandleFunc("/api/memory/{factory}", apiRoute(func(w http.ResponseWriter, r *http.Request) {
		files := memory.Files{Store: store, Factory: r.PathValue("factory")}
		q := r.URL.Query()
		path := q.Get("path")
		var result any
		var err error
		switch r.Method {
		case "GET":
			if path == "" {
				result, err = files.List(q.Get("directory"))
				break
			}
			n := 0
			if raw := q.Get("version"); raw != "" {
				n, err = strconv.Atoi(raw)
				if err != nil || n < 1 {
					http.Error(w, "invalid version", 400)
					return
				}
			}
			result, err = files.Read(memory.Ref{Path: path, Version: n})
		case "POST":
			var edit struct {
				Path     string
				Content  *string
				Expected *int
			}
			if !readEdit(w, r, &edit) {
				return
			}
			if edit.Content == nil || edit.Expected == nil || strings.HasPrefix(edit.Path, "reviews/") {
				http.Error(w, "supply a writable path, content and expected revision", 400)
				return
			}
			result, err = files.Write(edit.Path, *edit.Content, "human", *edit.Expected)
		default:
			http.Error(w, "use GET or POST", 405)
			return
		}
		if errors.Is(err, fs.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		if errors.Is(err, memory.ErrInvalid) {
			http.Error(w, err.Error(), 400)
			return
		}
		sendJSON(w, result, err)
	}, "GET", "POST"))
}
