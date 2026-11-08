package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
)

func TestLocalizedNotes(t *testing.T) {
	notes := "### 中文\n\n- 新功能\n\n### 修复\n- 修复内容\n\n### English\n\n- New feature\n\n### Fixed\n- Fixed things"

	zh := localizedNotes(notes, "zh-Hans")
	if !strings.Contains(zh, "新功能") || !strings.Contains(zh, "修复内容") {
		t.Errorf("Chinese notes = %q, want the Chinese section", zh)
	}
	if strings.Contains(zh, "New feature") || strings.Contains(zh, "Fixed things") {
		t.Errorf("Chinese notes = %q, want no English", zh)
	}

	en := localizedNotes(notes, "en")
	if !strings.Contains(en, "New feature") || !strings.Contains(en, "Fixed things") {
		t.Errorf("English notes = %q, want the English section", en)
	}
	if strings.Contains(en, "新功能") || strings.Contains(en, "修复内容") {
		t.Errorf("English notes = %q, want no Chinese", en)
	}

	// Notes from releases before the changelog have no language headings.
	old := "- old notes"
	if got := localizedNotes(old, "en"); got != old {
		t.Errorf("notes without headings = %q, want %q", got, old)
	}

	// Without the chosen language, the notes stay whole.
	onlyChinese := "### 中文\n\n- 只有中文"
	if got := localizedNotes(onlyChinese, "en"); got != onlyChinese {
		t.Errorf("notes without English = %q, want %q", got, onlyChinese)
	}
}

func TestLocalizedFeedManifest(t *testing.T) {
	target := runtime.GOOS + "-" + runtime.GOARCH
	var requested []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested = append(requested, r.URL.Path)
		switch r.URL.Path {
		case "/update-" + target + ".json":
			fmt.Fprint(w, `{"version":"1.2.0","notes":"### 中文\n- 中文说明\n### English\n- English notes","url":"asset","size":7,"signature":"sig"}`)
		case "/asset":
			fmt.Fprint(w, "payload")
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	for _, locale := range []string{"zh-Hans", "en"} {
		f := localizedFeed{base: srv.URL + "/", locale: func() string { return locale }}
		body, err := f.Manifest(context.Background(), false)
		if err != nil {
			t.Fatalf("%s: %v", locale, err)
		}
		data, err := io.ReadAll(body)
		body.Close()
		if err != nil {
			t.Fatalf("%s: reading the manifest: %v", locale, err)
		}
		var manifest map[string]json.RawMessage
		if err := json.Unmarshal(data, &manifest); err != nil {
			t.Fatalf("%s: %v", locale, err)
		}
		var notes string
		if err := json.Unmarshal(manifest["notes"], &notes); err != nil {
			t.Fatalf("%s: %v", locale, err)
		}
		want := "- English notes"
		if locale == "zh-Hans" {
			want = "- 中文说明"
		}
		if notes != want {
			t.Errorf("%s: notes = %q, want %q", locale, notes, want)
		}
		// The other fields survive the rewrite.
		for _, key := range []string{"version", "url", "size", "signature"} {
			if _, ok := manifest[key]; !ok {
				t.Errorf("%s: manifest lost %q", locale, key)
			}
		}

		asset, err := f.Open(context.Background(), "asset")
		if err != nil {
			t.Fatalf("%s: opening the asset: %v", locale, err)
		}
		payload, err := io.ReadAll(asset)
		asset.Close()
		if err != nil || string(payload) != "payload" {
			t.Errorf("%s: asset = %q, %v", locale, payload, err)
		}
	}

	wantPath := "/update-" + target + ".json"
	if len(requested) == 0 || requested[0] != wantPath {
		t.Errorf("requested %v, want %q first", requested, wantPath)
	}
}
