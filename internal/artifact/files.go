// Package artifact stores immutable files used as evidence in goal conversations.
package artifact

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/idristarwala/agent-factory/internal/llm"
	"github.com/idristarwala/agent-factory/internal/revision"
)

type Ref struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
}

type File struct {
	Passages []passage `json:"passages,omitempty"`
	Name     string    `json:"name"`
	MIME     string    `json:"mime"`
	SHA256   string    `json:"sha256"`
	Data     []byte    `json:"data"`
}

func Save(s revision.Store, ref Ref, name string, data []byte) (Ref, error) {
	if name == "" || len(name) > 255 || strings.ContainsAny(name, "/\\") || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return Ref{}, fmt.Errorf("invalid filename")
	}
	if len(data) == 0 || len(data) > 512<<10 {
		return Ref{}, fmt.Errorf("attach a nonempty file up to 512 KiB")
	}
	var passages []passage
	var err error
	mime := http.DetectContentType(data)
	switch strings.ToLower(filepath.Ext(name)) {
	case ".txt", ".md":
		if len(data) > 32<<10 || !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0 {
			return Ref{}, fmt.Errorf("text files must be UTF-8 and at most 32 KiB")
		}
		mime = "text/plain"
	case ".pdf":
		passages, err = pdfPassages(data)
		mime = "application/pdf"
	case ".docx":
		passages, err = wordPassages(data)
		mime = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".png", ".jpg", ".jpeg", ".webp":
		if mime != "image/png" && mime != "image/jpeg" && mime != "image/webp" {
			return Ref{}, fmt.Errorf("unrecognized image content")
		}
	default:
		return Ref{}, fmt.Errorf("attach PDF, DOCX, TXT, Markdown, PNG, JPEG or WebP")
	}
	if err != nil {
		return Ref{}, err
	}
	if ref.ID == "" && ref.Version == 0 {
		ref.ID = uuid.NewString()
	} else if uuid.Validate(ref.ID) != nil || ref.Version < 1 {
		return Ref{}, fmt.Errorf("replacement requires an artifact ID and its current revision")
	}
	b, err := json.Marshal(File{Passages: passages, Name: name, MIME: mime, SHA256: fmt.Sprintf("%x", sha256.Sum256(data)), Data: data})
	if err != nil {
		return Ref{}, err
	}
	d, err := revision.SaveDocument(s, "chat-artifact/"+ref.ID, ref.Version, "human", b)
	if err != nil {
		return Ref{}, err
	}
	ref.Version = d.Version
	return ref, nil
}

type Evidence struct {
	Text   string
	Images []llm.ImagePart
}

func Read(s revision.Store, refs []Ref) (Evidence, error) {
	var out Evidence
	if len(refs) > 4 {
		return out, fmt.Errorf("select at most four chat artifacts")
	}
	seen := map[string]bool{}
	var context strings.Builder
	for _, r := range refs {
		if uuid.Validate(r.ID) != nil || r.Version < 1 || seen[r.ID] {
			return out, fmt.Errorf("invalid or duplicate artifact reference")
		}
		seen[r.ID] = true
		h, err := revision.Documents(s, "chat-artifact/"+r.ID)
		if err != nil {
			return out, err
		}
		if r.Version > len(h) {
			return out, fmt.Errorf("unknown artifact revision")
		}
		var f File
		if err = json.Unmarshal(h[r.Version-1].Data, &f); err != nil {
			return out, err
		}
		fmt.Fprintf(&context, "\nAttachment %q revision %d (sha256 %s)", f.Name, r.Version, f.SHA256)
		if len(f.Passages) > 0 {
			for _, part := range f.Passages {
				fmt.Fprintf(&context, "\n[%s, %s] %s\n", f.Name, part.Location, part.Text)
			}
		} else if f.MIME == "text/plain" {
			fmt.Fprintf(&context, ":\n%s\n", f.Data)
		} else {
			out.Images = append(out.Images, llm.ImagePart{Mime: f.MIME, Base64: base64.StdEncoding.EncodeToString(f.Data)})
			fmt.Fprintf(&context, " is attached image %d.\n", len(out.Images))
		}
	}
	out.Text = context.String()
	return out, nil
}
