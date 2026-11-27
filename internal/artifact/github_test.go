package artifact

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/idristarwala/agent-factory/internal/revision"
)

type githubTransport func(*http.Request) (*http.Response, error)

func (f githubTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGitHubSnapshotAndCredentialScope(t *testing.T) {
	s := revision.NewMemory()
	title := "Review the brief"
	g := GitHub{Token: "fixture", Repositories: []string{"Team/Selected"}}
	g.transport = githubTransport(func(r *http.Request) (*http.Response, error) {
		allowed := strings.HasPrefix(r.URL.Path, "/repos/team/selected/")
		if r.URL.Host != "api.github.com" || (r.Header.Get("Authorization") == "Bearer fixture") != allowed {
			t.Fatal(r.URL)
		}
		body := fmt.Sprintf(`{"title":%q,"state":"open","body":"Two requirements","pull_request":{}}`, title)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}, nil
	})
	url := "https://github.com/team/selected/pull/4"
	first, err := g.Import(t.Context(), s, Ref{}, url)
	if err != nil {
		t.Fatal(err)
	}
	title = "Revised brief"
	second, err := g.Import(t.Context(), s, first.Ref, url)
	if err != nil || second.Version != 2 {
		t.Fatal(second, err)
	}
	old, err := Read(s, []Ref{first.Ref})
	if err != nil || !strings.Contains(old.Text, "Review the brief") || !strings.Contains(old.Text, url) || strings.Contains(old.Text, "fixture") {
		t.Fatal(old, err)
	}
	if _, err = g.Import(t.Context(), s, Ref{}, "https://github.com/team/public/pull/4"); err != nil {
		t.Fatal(err)
	}
	if _, err = g.Import(t.Context(), s, Ref{}, "https://example.com/team/selected/pull/4"); err == nil {
		t.Fatal("unexpected link accepted")
	}
}

func TestGitHubFilePinnedBeforeRetrieval(t *testing.T) {
	sha := strings.Repeat("1", 40)
	g := GitHub{transport: githubTransport(func(r *http.Request) (*http.Response, error) {
		var body string
		switch r.URL.RequestURI() {
		case "/repos/a/b/commits/main":
			body = sha
		case "/repos/a/b/contents/pkg/example.go?ref=" + sha:
			body = `{"type":"file","path":"pkg/example.go","encoding":"base64","content":"cGFja2FnZSBkZW1v"}`
		default:
			t.Fatal(r.URL)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}, nil
	})}
	s := revision.NewMemory()
	f, err := g.Import(t.Context(), s, Ref{}, "https://github.com/a/b/blob/main/pkg/example.go")
	if err != nil || !strings.Contains(f.Source, sha) {
		t.Fatal(f, err)
	}
	e, err := Read(s, []Ref{f.Ref})
	if err != nil || !strings.Contains(e.Text, "package demo") || !strings.Contains(e.Text, f.Source) {
		t.Fatal(e, err)
	}
}
