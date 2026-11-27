package factory

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/idristarwala/agent-factory/internal/artifact"
	"github.com/idristarwala/agent-factory/internal/llm"
	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/schema"
	"github.com/idristarwala/agent-factory/internal/tools"
)

type BrowserTool struct {
	Name         string   `json:"name"`
	Revision     int      `json:"revision"`
	PackRevision int      `json:"pack_revision"`
	Origins      []string `json:"origins"`
	URLField     string   `json:"url_field"`
	Screenshot   bool     `json:"screenshot"`
	TimeoutMS    int      `json:"timeout_ms"`
	MaxTextBytes int      `json:"max_text_bytes"`
}

type BrowserCapture struct {
	ID           string    `json:"id"`
	Tool         string    `json:"tool"`
	ToolRevision int       `json:"tool_revision"`
	PackRevision int       `json:"pack_revision"`
	Source       string    `json:"source"`
	URL          string    `json:"url"`
	CapturedAt   time.Time `json:"captured_at"`
	Text         string    `json:"text"`
	Image        []byte    `json:"image,omitempty"`
}

func origin(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") || strings.Contains(raw, "\\") {
		return "", fmt.Errorf("expected an HTTP(S) capture URL without credentials")
	}
	host := strings.ToLower(u.Host)
	if (u.Scheme == "https" && u.Port() == "443") || (u.Scheme == "http" && u.Port() == "80") {
		host = strings.TrimSuffix(host, ":"+u.Port())
	}
	return u.Scheme + "://" + host, nil
}

func pinBrowserTool(store revision.Store, pack, name string) (*BrowserTool, error) {
	var def struct {
		BrowserTool
		Kind       string
		SideEffect string `json:"side_effect"`
	}
	def.TimeoutMS = 10000
	def.MaxTextBytes = 8192
	revision, err := latestProposalDefinition(store, "tool/"+name, &def)
	if err != nil {
		return nil, err
	}
	if def.Kind != "playwright" || def.SideEffect != "read-only" || def.URLField == "" || len(def.Origins) == 0 || def.TimeoutMS < 1 || def.TimeoutMS > 60000 || def.MaxTextBytes < 1 || def.MaxTextBytes > 32768 {
		return nil, fmt.Errorf("invalid read-only Playwright definition %q", name)
	}
	for i, v := range def.Origins {
		o, err := origin(v)
		u, parseErr := url.Parse(v)
		if err != nil || parseErr != nil || (u.EscapedPath() != "" && u.EscapedPath() != "/") || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
			return nil, fmt.Errorf("expected exact HTTP(S) origin")
		}
		def.Origins[i] = o
	}
	var policy struct{ Grants []tools.Grant }
	version, err := latestProposalDefinition(store, "pack/"+pack, &policy)
	if err != nil {
		return nil, err
	}
	if !tools.CheckGrant(policy.Grants, pack, name, "read-only") {
		return nil, fmt.Errorf("pack %q has no read-only grant for %q", pack, name)
	}
	def.Name = name
	def.Revision = revision
	def.PackRevision = version
	return &def.BrowserTool, nil
}

func (b BrowserTool) capture(store revision.Store, record schema.Entity, deadline time.Time) (BrowserCapture, error) {
	var capture BrowserCapture
	raw, _ := record.Attributes[b.URLField].(string)
	o, err := origin(raw)
	if err != nil || !slices.Contains(b.Origins, o) {
		return capture, fmt.Errorf("browser capture source origin denied")
	}
	script := os.Getenv("FACTORY_PLAYWRIGHT_SCRIPT")
	if !filepath.IsAbs(script) {
		return capture, fmt.Errorf("browser capture requires an absolute FACTORY_PLAYWRIGHT_SCRIPT")
	}
	request := struct {
		BrowserTool
		URL string `json:"url"`
	}{b, raw}
	input, err := json.Marshal(request)
	if err != nil {
		return capture, err
	}
	timeout := min(time.Until(deadline), time.Duration(b.TimeoutMS)*time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "node", script)
	cmd.Env = []string{"PATH=" + os.Getenv("PATH")}
	for _, key := range []string{"TMPDIR", "SystemRoot", "PLAYWRIGHT_BROWSERS_PATH", "FACTORY_CHROMIUM_EXECUTABLE"} {
		if value := os.Getenv(key); value != "" {
			cmd.Env = append(cmd.Env, key+"="+value)
		}
	}
	cmd.Stdin = bytes.NewReader(input)
	cmd.WaitDelay = time.Second
	data, err := cmd.Output()
	if err != nil {
		return capture, fmt.Errorf("browser capture failed for tool %q: %w", b.Name, err)
	}
	var result struct {
		URL, Text string
		Image     []byte
	}
	if len(data) > 1<<20 || json.Unmarshal(data, &result) != nil {
		return capture, fmt.Errorf("invalid capture response")
	}
	finalOrigin, err := origin(result.URL)
	if err != nil || !slices.Contains(b.Origins, finalOrigin) || len(result.Text) > b.MaxTextBytes || len(result.Image) > 512<<10 || (len(result.Image) > 0) != b.Screenshot || (b.Screenshot && http.DetectContentType(result.Image) != "image/jpeg") {
		return capture, fmt.Errorf("capture exceeds configured scope or limits")
	}
	capture = BrowserCapture{ID: uuid.NewString(), Tool: b.Name, ToolRevision: b.Revision, PackRevision: b.PackRevision, Source: raw, URL: result.URL, CapturedAt: time.Now().UTC(), Text: result.Text, Image: result.Image}
	data, err = json.Marshal(capture)
	if err != nil {
		return capture, err
	}
	_, err = revision.SaveDocument(store, "browser-capture/"+capture.ID, 0, "steward", data)
	return capture, err
}

// ReadBrowserCapture reads the original immutable capture, even after source changes.
func ReadBrowserCapture(store revision.Store, id string) (BrowserCapture, error) {
	var c BrowserCapture
	if uuid.Validate(id) != nil {
		return c, ErrNotFound
	}
	docs, err := revision.Documents(store, "browser-capture/"+id)
	if err != nil {
		return c, err
	}
	if len(docs) == 0 {
		return c, ErrNotFound
	}
	err = json.Unmarshal(docs[0].Data, &c)
	return c, err
}

func (c BrowserCapture) evidence(base artifact.Evidence) artifact.Evidence {
	base.Text += fmt.Sprintf("\nBrowser capture %s, tool %s r%d, %s at %s. Rendered page data:\n%s", c.ID, c.Tool, c.ToolRevision, c.URL, c.CapturedAt.Format(time.RFC3339Nano), c.Text)
	if len(c.Image) > 0 {
		base.Images = append(slices.Clone(base.Images), llm.ImagePart{Mime: "image/jpeg", Base64: base64.StdEncoding.EncodeToString(c.Image)})
		base.Text += fmt.Sprintf("\nCapture screenshot image %d (sha256 %x)", len(base.Images), sha256.Sum256(c.Image))
	}
	return base
}
