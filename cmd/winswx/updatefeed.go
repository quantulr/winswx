package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strings"

	"github.com/egoist/mygo/netproxy"
)

// updateFeed is the download directory of the latest release that
// mygo.json configures. localizedFeed reads the manifest from it instead
// of the feed MyGo resolves, so that it can choose the notes' language.
const updateFeed = "https://github.com/quantulr/winswx/releases/latest/download/"

// The headings that divide the release notes in CHANGELOG.md by language.
const (
	notesChinese = "### 中文"
	notesEnglish = "### English"
)

// updateClient reaches the update feed through the system's proxy.
var updateClient = &http.Client{Transport: netproxy.Transport()}

// localizedFeed serves the release manifest with the notes of the app's
// language. The manifest is not signed itself; the archive and delta
// signatures are, so changing the notes does not affect what is installed.
type localizedFeed struct {
	base   string        // the directory that holds the manifest and files
	locale func() string // the language the app shows
}

func (f localizedFeed) Manifest(ctx context.Context, _ bool) (io.ReadCloser, error) {
	target := runtime.GOOS + "-" + runtime.GOARCH
	body, err := get(ctx, f.base+"update-"+target+".json")
	if err != nil {
		return nil, err
	}
	defer body.Close()
	data, err := io.ReadAll(io.LimitReader(body, 1<<20))
	if err != nil {
		return nil, err
	}
	var manifest map[string]json.RawMessage
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	var notes string
	if raw, ok := manifest["notes"]; ok {
		if err := json.Unmarshal(raw, &notes); err != nil {
			return nil, err
		}
	}
	if localized := localizedNotes(notes, f.locale()); localized != notes {
		raw, err := json.Marshal(localized)
		if err != nil {
			return nil, err
		}
		manifest["notes"] = raw
	}
	out, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(out)), nil
}

func (f localizedFeed) Open(ctx context.Context, rawURL string) (io.ReadCloser, error) {
	base, err := url.Parse(f.base)
	if err != nil {
		return nil, err
	}
	ref, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	return get(ctx, base.ResolveReference(ref).String())
}

// get fetches a URL over HTTPS, or HTTP from the loopback interface for
// tests, through the system's proxy as MyGo's own updater does.
func get(ctx context.Context, rawURL string) (io.ReadCloser, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && isLoopback(u.Hostname())) {
		return nil, fmt.Errorf("refusing to download %s: updates need HTTPS", rawURL)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "mygo-updater")
	resp, err := updateClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("%s: %s", rawURL, resp.Status)
	}
	return resp.Body, nil
}

func isLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// localizedNotes returns the part of notes under the heading of locale:
// "### 中文" for a Chinese locale, "### English" otherwise. Notes without
// that heading, as releases before the changelog had, are returned as they
// are.
func localizedNotes(notes, locale string) string {
	heading, other := notesEnglish, notesChinese
	if strings.HasPrefix(strings.ToLower(locale), "zh") {
		heading, other = notesChinese, notesEnglish
	}
	lines := strings.Split(strings.ReplaceAll(notes, "\r\n", "\n"), "\n")
	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == heading {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return notes
	}
	end := len(lines)
	for i := start; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == other {
			end = i
			break
		}
	}
	text := strings.TrimSpace(strings.Join(lines[start:end], "\n"))
	if text == "" {
		return notes
	}
	return text
}
