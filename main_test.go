package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// The view runs without a window in tests, which build it as a user would
// see it.
func TestLoadServices(t *testing.T) {
	userData := t.TempDir()
	mygo.App.SetPath(mygo.PathUserData, userData)
	t.Cleanup(func() { mygo.App.SetPath(mygo.PathUserData, "") })

	a := &app{}
	// The first load makes the services directory.
	a.load()
	servicesDir := filepath.Join(userData, "services")
	if info, err := os.Stat(servicesDir); err != nil || !info.IsDir() {
		t.Fatalf("services directory not created: %v", err)
	}
	for _, name := range []string{"caddy", "mihomo"} {
		if err := os.Mkdir(filepath.Join(servicesDir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// A file in the directory is not a service.
	if err := os.WriteFile(filepath.Join(servicesDir, "README"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	a.load()
	if len(a.services) != 2 || a.services[0].name != "caddy" || a.services[1].name != "mihomo" {
		t.Fatalf("services %v, want [caddy mihomo]", a.services)
	}

	tt := ui.NewTester(a.view, 480, 360)
	for _, s := range a.services {
		if !tt.HasText(s.name) {
			t.Errorf("texts %q do not show service %q", tt.Texts(), s.name)
		}
		if !tt.HasText(string(s.status)) {
			t.Errorf("texts %q do not show the status of %q: %q", tt.Texts(), s.name, s.status)
		}
	}
	if a.winsw == "" {
		t.Error("the bundled winsw.exe was not found")
	}
}

func TestWinswBundled(t *testing.T) {
	exe, err := winswExe()
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(exe)
	if err != nil {
		t.Fatalf("bundled winsw at %s: %v", exe, err)
	}
	if info.Size() == 0 {
		t.Fatalf("bundled winsw at %s is empty", exe)
	}
}

func TestViewShowsServiceActions(t *testing.T) {
	a := &app{services: []service{{name: "caddy", status: statusStopped}}}
	tt := ui.NewTester(a.view, 480, 360)
	for _, label := range []string{"caddy", "Stopped", "Start", "More"} {
		if !tt.HasText(label) {
			t.Errorf("texts %q do not show %q", tt.Texts(), label)
		}
	}

	a = &app{services: []service{{name: "new", status: statusNotInstalled}}}
	tt = ui.NewTester(a.view, 480, 360)
	for _, label := range []string{"new", "Not installed", "Install"} {
		if !tt.HasText(label) {
			t.Errorf("texts %q do not show %q", tt.Texts(), label)
		}
	}
	if tt.HasText("More") {
		t.Errorf("texts %q show More for a service that is not installed", tt.Texts())
	}
}

func TestServiceConfig(t *testing.T) {
	dir := t.TempDir()
	write := func(path string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// The XML named after the directory wins.
	named := filepath.Join(dir, "caddy", "caddy.xml")
	write(named)
	write(filepath.Join(dir, "caddy", "other.xml"))
	if got := serviceConfig(dir, "caddy"); got != named {
		t.Errorf("serviceConfig = %q, want %q", got, named)
	}

	// Otherwise the only XML is the configuration.
	only := filepath.Join(dir, "clash", "clash-config.xml")
	write(only)
	if got := serviceConfig(dir, "clash"); got != only {
		t.Errorf("serviceConfig = %q, want %q", got, only)
	}

	// More than one XML is ambiguous.
	write(filepath.Join(dir, "clash", "second.xml"))
	if got := serviceConfig(dir, "clash"); got != "" {
		t.Errorf("serviceConfig with two XMLs = %q, want \"\"", got)
	}

	// A service without XML has nothing to query.
	if got := serviceConfig(dir, "missing"); got != "" {
		t.Errorf("serviceConfig of a missing directory = %q, want \"\"", got)
	}
}

func TestParseStatus(t *testing.T) {
	cases := map[string]serviceStatus{
		"Active (running)":    statusRunning,
		"Active (starting)":   statusStarting,
		"Active (stopping)":   statusStopping,
		"Active (pausing)":    statusPaused,
		"Active (paused)":     statusPaused,
		"Active (continuing)": statusPaused,
		"Inactive (stopped)":  statusStopped,
		"NonExistent":         statusNotInstalled,
		"":                    statusUnknown,
		"something else":      statusUnknown,
	}
	for out, want := range cases {
		if got := parseStatus(out); got != want {
			t.Errorf("parseStatus(%q) = %q, want %q", out, got, want)
		}
	}
}

// TestCardWidthFixed checks that cards keep their width as the window
// resizes, while the grid adds or drops columns, and that their actions
// stay inside them.
func TestCardWidthFixed(t *testing.T) {
	a := &app{services: []service{
		{name: "caddy", status: statusRunning},
		{name: "mihomo", status: statusRunning},
		{name: "clash", status: statusRunning},
	}}
	tt := ui.NewTester(a.view, 480, 360)
	for _, width := range []int{480, 700, 900, 1200, 400, 320} {
		tt.SetSize(width, 360)
		card, ok := tt.Find("caddy") // the card is labeled with its service
		if !ok {
			t.Fatalf("width %d: card not found", width)
		}
		if card.W != cardWidth {
			t.Errorf("width %d: card is %v wide, want %v", width, card.W, cardWidth)
		}
		// The card's content ends inside its 1 DIP border and 14 DIP padding.
		limit := card.X + card.W - 15
		for _, label := range []string{"Stop", "More"} {
			r, ok := tt.Find(label)
			if !ok {
				t.Fatalf("width %d: %s not found", width, label)
			}
			// The text sits 7 DIP of padding and 1 DIP of border from the
			// button's edge.
			if right := r.X + r.W + 8; right > limit {
				t.Errorf("width %d: %s ends at %.1f, past the card content at %.1f", width, label, right, limit)
			}
		}
	}
}
