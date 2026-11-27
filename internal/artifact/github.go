package artifact

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/idristarwala/agent-factory/internal/revision"
)

var repoPart = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)
var commitHash = regexp.MustCompile(`^[a-fA-F0-9]{40}$`)

type GitHub struct {
	transport    http.RoundTripper
	Token        string
	Repositories []string
}

type GitHubFile struct {
	Ref
	Name   string `json:"name"`
	Source string `json:"source"`
}

// Import snapshots a user-selected resource from the fixed GitHub API origin.
func (g GitHub) Import(ctx context.Context, s revision.Store, ref Ref, link string) (GitHubFile, error) {
	empty := GitHubFile{}
	u, err := url.Parse(link)
	if err != nil || u.Scheme != "https" || u.Host != "github.com" || u.User != nil || u.RawQuery != "" || len(link) > 2048 {
		return empty, fmt.Errorf("use a github.com repository, file, issue or pull request URL")
	}
	if u.Fragment != "" {
		return empty, fmt.Errorf("attach the whole resource by removing the URL fragment")
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 || !repoPart.MatchString(parts[0]) || !repoPart.MatchString(parts[1]) {
		return empty, fmt.Errorf("invalid GitHub repository")
	}
	for _, p := range parts {
		if p == "" || p == "." || p == ".." || strings.Contains(p, "\\") || strings.IndexFunc(p, unicode.IsControl) >= 0 {
			return empty, fmt.Errorf("invalid GitHub path")
		}
	}
	repo := parts[0] + "/" + parts[1]
	base := "https://api.github.com/repos/" + repo
	canonical := "https://github.com/" + repo
	var token string
	for _, allowed := range g.Repositories {
		if strings.EqualFold(strings.TrimSpace(allowed), repo) {
			token = g.Token
			break
		}
	}
	client := &http.Client{Transport: g.transport, Timeout: 12 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	get := func(endpoint, accept string) ([]byte, error) {
		req, err := http.NewRequestWithContext(ctx, "GET", base+endpoint, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", accept)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		res, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("GitHub request failed")
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			return nil, fmt.Errorf("GitHub returned HTTP %d; check the link and repository access", res.StatusCode)
		}
		data, err := io.ReadAll(io.LimitReader(res.Body, (1<<20)+1))
		if err != nil || len(data) > 1<<20 {
			return nil, fmt.Errorf("GitHub response exceeds limit or is unreadable")
		}
		return data, nil
	}
	var name string
	var data []byte
	switch {
	case len(parts) == 2 || (len(parts) >= 5 && parts[2] == "blob"):
		branch := "HEAD"
		endpoint := "/readme"
		if len(parts) > 2 {
			branch = parts[3]
			escaped := make([]string, 0, len(parts)-4)
			for _, p := range parts[4:] {
				escaped = append(escaped, url.PathEscape(p))
			}
			endpoint = "/contents/" + strings.Join(escaped, "/")
		}
		sha := branch
		if !commitHash.MatchString(sha) {
			digest, err := get("/commits/"+url.PathEscape(branch), "application/vnd.github.sha")
			if err != nil {
				return empty, err
			}
			sha = strings.TrimSpace(string(digest))
		}
		if !commitHash.MatchString(sha) {
			return empty, fmt.Errorf("could not resolve the GitHub commit")
		}
		body, err := get(endpoint+"?ref="+sha, "application/vnd.github+json")
		if err != nil {
			return empty, err
		}
		var f struct {
			Type, Encoding, Path string
			Content              []byte
		}
		if json.Unmarshal(body, &f) != nil || f.Type != "file" || f.Encoding != "base64" || f.Path == "" {
			return empty, fmt.Errorf("select a supported repository file")
		}
		name = path.Base(f.Path)
		canonical += (&url.URL{Path: "/blob/" + sha + "/" + f.Path}).EscapedPath()
		data = f.Content
	case len(parts) == 4 && (parts[2] == "issues" || parts[2] == "pull"):
		n, err := strconv.Atoi(parts[3])
		if err != nil || n < 1 {
			return empty, fmt.Errorf("invalid issue or pull request number")
		}
		body, err := get("/issues/"+strconv.Itoa(n), "application/vnd.github+json")
		if err != nil {
			return empty, err
		}
		var item struct {
			Title, Body, State string
			PullRequest        *struct{} `json:"pull_request"`
		}
		if json.Unmarshal(body, &item) != nil || item.Title == "" || (item.PullRequest != nil) != (parts[2] == "pull") {
			return empty, fmt.Errorf("link does not match the GitHub resource")
		}
		name = parts[1] + "-" + parts[2] + "-" + strconv.Itoa(n) + ".md"
		canonical += "/" + parts[2] + "/" + strconv.Itoa(n)
		data = []byte(fmt.Sprintf("# %s\nState: %s\n\n%s", item.Title, item.State, item.Body))
	default:
		return empty, fmt.Errorf("select a repository README, blob, issue or pull request")
	}
	saved, err := save(s, ref, name, data, canonical)
	return GitHubFile{Ref: saved, Name: name, Source: canonical}, err
}
