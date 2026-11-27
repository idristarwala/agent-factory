package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

// workspaceUI serves the public application shell and its compiled assets.
// The API keeps its existing route-specific authorization.
func workspaceUI(mux *http.ServeMux, dir string) error {
	if dir == "" {
		return nil
	}
	index, err := os.OpenInRoot(dir, "index.html")
	if err != nil {
		return fmt.Errorf("open workspace index: %w", err)
	}
	info, err := index.Stat()
	index.Close()
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("expected a regular index.html in workspace directory %q", dir)
	}
	serve := func(w http.ResponseWriter, r *http.Request, name string) {
		file, err := os.OpenInRoot(dir, name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil || !info.Mode().IsRegular() {
			http.NotFound(w, r)
			return
		}
		http.ServeContent(w, r, name, info.ModTime(), file)
	}
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		serve(w, r, "index.html")
	})
	mux.HandleFunc("GET /assets/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if strings.HasPrefix(name, ".") || strings.ContainsAny(name, "/\\") {
			http.NotFound(w, r)
			return
		}
		serve(w, r, "assets/"+name)
	})
	return nil
}
