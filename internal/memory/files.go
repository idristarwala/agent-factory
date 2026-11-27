package memory

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"time"
	"unicode"

	"github.com/idristarwala/agent-factory/internal/revision"
)

var ErrInvalid = errors.New("invalid memory file")

type Ref struct {
	Path    string `json:"path"`
	Version int    `json:"version"`
}
type File struct {
	Ref
	Author  string    `json:"author"`
	Updated time.Time `json:"updated"`
	Content string    `json:"content"`
}
type Files struct {
	Store   revision.Store
	Factory string
}

func validPath(p string) bool {
	return len(p) <= 512 && fs.ValidPath(p) && p != "." && !strings.Contains(p, "\\") && strings.IndexFunc(p, unicode.IsControl) < 0
}
func (f Files) prefix() string { return base64.RawURLEncoding.EncodeToString([]byte(f.Factory)) + "/" }
func (f Files) Read(ref Ref) (File, error) {
	if f.Factory == "" || !validPath(ref.Path) || ref.Version < 0 {
		return File{}, ErrInvalid
	}
	rows, err := f.Store.Read(revision.MemoryFile, f.prefix()+ref.Path)
	if err != nil {
		return File{}, err
	}
	n := ref.Version
	if n == 0 {
		n = len(rows)
	}
	if n < 1 || n > len(rows) {
		return File{}, fs.ErrNotExist
	}
	var file File
	err = json.Unmarshal(rows[n-1].Data, &file)
	return file, err
}
func (f Files) Write(path, content, author string, expected int) (File, error) {
	if f.Factory == "" || !validPath(path) || author == "" || len(content) > 32<<10 || expected < 0 {
		return File{}, ErrInvalid
	}
	prev, err := f.Read(Ref{Path: path})
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return File{}, err
	}
	if prev.Version != expected {
		return File{}, revision.ErrConflict
	}
	if expected > 0 && prev.Content == content {
		return prev, nil
	}
	file := File{Ref: Ref{Path: path, Version: expected + 1}, Content: content, Author: author, Updated: time.Now().UTC()}
	data, err := json.Marshal(file)
	if err != nil {
		return File{}, err
	}
	err = f.Store.Insert(revision.MemoryFile, revision.Snapshot{ID: f.prefix() + path, Version: file.Version, Data: data})
	return file, err
}
func (f Files) List(directory string) ([]Ref, error) {
	if f.Factory == "" || (directory != "" && !validPath(directory)) {
		return nil, ErrInvalid
	}
	prefix := f.prefix()
	if directory != "" {
		prefix += directory + "/"
	}
	rows, err := f.Store.Read(revision.MemoryFile, "")
	if err != nil {
		return nil, err
	}
	out := []Ref{}
	for _, r := range rows {
		if !strings.HasPrefix(r.ID, prefix) {
			continue
		}
		path := strings.TrimPrefix(r.ID, f.prefix())
		ref := Ref{path, r.Version}
		if len(out) > 0 && out[len(out)-1].Path == path {
			out[len(out)-1] = ref
		} else {
			out = append(out, ref)
		}
	}
	return out, nil
}
func (f Files) Pin(refs []Ref) ([]File, error) {
	if len(refs) > 16 {
		return nil, fmt.Errorf("select up to 16 memory files")
	}
	var files []File
	size := 0
	seen := map[string]bool{}
	for _, ref := range refs {
		if seen[ref.Path] {
			return nil, fmt.Errorf("duplicate memory path")
		}
		seen[ref.Path] = true
		file, err := f.Read(ref)
		if err != nil {
			return nil, err
		}
		size += len(file.Content)
		if size > 64<<10 {
			return nil, fmt.Errorf("selected memory exceeds 64 KiB")
		}
		files = append(files, file)
	}
	return files, nil
}
