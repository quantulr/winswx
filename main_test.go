package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"mygo/locales"
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
		if !tt.HasText(statusLabel(s.status)) {
			t.Errorf("texts %q do not show the status of %q: %q", tt.Texts(), s.name, s.status)
		}
	}
	if a.winsw == "" {
		t.Error("the bundled winsw.exe was not found; run `go run ./scripts/fetch-winsw` first")
	}
}

func TestWinswBundled(t *testing.T) {
	exe, err := winswExe()
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(exe)
	if err != nil {
		t.Fatalf("bundled winsw at %s: %v (run `go run ./scripts/fetch-winsw` first)", exe, err)
	}
	if info.Size() == 0 {
		t.Fatalf("bundled winsw at %s is empty", exe)
	}
}

func TestViewShowsServiceActions(t *testing.T) {
	a := &app{services: []service{{name: "caddy", status: statusStopped}}}
	tt := ui.NewTester(a.view, 480, 360)
	for _, label := range []string{"caddy", statusLabel(statusStopped), locales.Start(), locales.More()} {
		if !tt.HasText(label) {
			t.Errorf("texts %q do not show %q", tt.Texts(), label)
		}
	}

	a = &app{services: []service{{name: "new", status: statusNotInstalled}}}
	tt = ui.NewTester(a.view, 480, 360)
	for _, label := range []string{"new", statusLabel(statusNotInstalled), locales.Install(), locales.More()} {
		if !tt.HasText(label) {
			t.Errorf("texts %q do not show %q", tt.Texts(), label)
		}
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
		for _, label := range []string{locales.Stop(), locales.More()} {
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

func TestButtonsUsePointerCursor(t *testing.T) {
	a := &app{services: []service{{name: "caddy", status: statusStopped}}}
	tt := ui.NewTester(a.view, 480, 360)
	for _, label := range []string{locales.Start(), locales.More(), locales.Refresh()} {
		r, ok := tt.Find(label)
		if !ok {
			t.Fatalf("%s not found", label)
		}
		tt.Move(r.X+r.W/2, r.Y+r.H/2)
		if got := tt.Cursor(); got != ui.CursorPointer {
			t.Errorf("cursor over %s = %v, want pointer", label, got)
		}
	}
}

// TestAddService creates the directory and WinSW configuration of a new
// service.
func TestAddService(t *testing.T) {
	userData := t.TempDir()
	mygo.App.SetPath(mygo.PathUserData, userData)
	t.Cleanup(func() { mygo.App.SetPath(mygo.PathUserData, "") })

	a := &app{}
	a.load()
	a.newName = "demo"
	a.newExe = `C:\Tools\demo.exe`
	a.newArgs = "-port 8080"
	if !a.addService() {
		t.Fatalf("addService: %s", a.dlgErr)
	}
	data, err := os.ReadFile(filepath.Join(userData, "services", "demo", "demo.xml"))
	if err != nil {
		t.Fatalf("reading the configuration: %v", err)
	}
	for _, want := range []string{
		"<id>demo</id>",
		`<executable>C:\Tools\demo.exe</executable>`,
		"<arguments>-port 8080</arguments>",
	} {
		if !strings.Contains(string(data), want) {
			t.Errorf("configuration %q does not contain %q", data, want)
		}
	}
	found := false
	for _, s := range a.services {
		if s.name == "demo" {
			found = true
		}
	}
	if !found {
		t.Errorf("services %v do not include demo", a.services)
	}

	// The same name twice is refused.
	a.newName = "demo"
	a.newExe = `C:\Tools\demo.exe`
	if a.addService() {
		t.Error("adding the same service twice was accepted")
	}
}

func TestAddServiceValidation(t *testing.T) {
	userData := t.TempDir()
	mygo.App.SetPath(mygo.PathUserData, userData)
	t.Cleanup(func() { mygo.App.SetPath(mygo.PathUserData, "") })

	a := &app{}
	a.load()
	for _, c := range []struct{ name, exe string }{
		{"", `C:\x.exe`},
		{"..", `C:\x.exe`},
		{`a/b`, `C:\x.exe`},
		{"CON", `C:\x.exe`},
		{" name", `C:\x.exe`},
		{"ok", ""},
	} {
		a.newName, a.newExe, a.newArgs, a.dlgErr = c.name, c.exe, "", ""
		a.addService()
		if a.dlgErr == "" {
			t.Errorf("addService(name %q, exe %q) was accepted", c.name, c.exe)
		}
	}
	entries, err := os.ReadDir(filepath.Join(userData, "services"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("rejected services created %v", entries)
	}
}

func TestServiceXML(t *testing.T) {
	data, err := serviceXML("demo", `C:\a&b.exe`, `-x "a & b" <c>`)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{"<id>demo</id>", `C:\a&amp;b.exe`, "&lt;c&gt;"} {
		if !strings.Contains(text, want) {
			t.Errorf("configuration %q does not contain %q", text, want)
		}
	}
}

func TestAddDialogOpens(t *testing.T) {
	a := &app{}
	tt := ui.NewTester(a.view, 480, 360)
	if err := tt.Click(locales.AddService()); err != nil {
		t.Fatalf("clicking Add Service: %v", err)
	}
	for _, want := range []string{locales.Browse(), locales.Cancel(), locales.Add()} {
		if !tt.HasText(want) {
			t.Errorf("texts %q do not show %q", tt.Texts(), want)
		}
	}
}

func TestCloseDialogAnimates(t *testing.T) {
	a := &app{dir: t.TempDir(), winsw: "winsw.exe"}
	tt := ui.NewTester(a.view, 480, 360)
	if err := tt.Click(locales.AddService()); err != nil {
		t.Fatal(err)
	}
	tt.Advance(400 * time.Millisecond)
	tt.Key(0, ui.KeyEscape)
	tt.Advance(40 * time.Millisecond)
	if a.dialog == dialogNone {
		t.Fatal("the dialog was removed before its closing animation")
	}
	tt.Advance(400 * time.Millisecond)
	if a.dialog != dialogNone {
		t.Fatal("the dialog stayed after its closing animation")
	}
}

func TestUpdateServiceXML(t *testing.T) {
	in := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<service>
  <id>old</id>
  <executable>C:\old.exe</executable>
  <arguments>-old</arguments>
  <log mode="roll-by-size"/>
  <serviceaccount>
    <username>LocalSystem</username>
  </serviceaccount>
</service>`)
	out, err := updateServiceXML(in, "new", `C:\new.exe`, "")
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	for _, want := range []string{
		"<id>new</id>",
		`<executable>C:\new.exe</executable>`,
		`mode="roll-by-size"`,
		"<username>LocalSystem</username>",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("configuration %q does not contain %q", text, want)
		}
	}
	if strings.Contains(text, "<arguments>") || strings.Contains(text, "-old") {
		t.Errorf("configuration %q kept the old arguments", text)
	}

	out, err = updateServiceXML(in, "new", `C:\new.exe`, "-x 1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "<arguments>-x 1</arguments>") {
		t.Errorf("configuration %q does not set the arguments", out)
	}
}

func TestEditService(t *testing.T) {
	userData := t.TempDir()
	mygo.App.SetPath(mygo.PathUserData, userData)
	t.Cleanup(func() { mygo.App.SetPath(mygo.PathUserData, "") })

	a := &app{}
	a.load()
	a.newName = "demo"
	a.newExe = `C:\old.exe`
	a.newArgs = "-old"
	if !a.addService() {
		t.Fatalf("addService: %s", a.dlgErr)
	}

	a.openEditDialog("demo")
	if a.dialog != dialogEdit || a.newName != "demo" || a.newExe != `C:\old.exe` || a.newArgs != "-old" {
		t.Fatalf("edit dialog got name %q, exe %q and args %q", a.newName, a.newExe, a.newArgs)
	}

	// The ID cannot change: a name typed into the dialog is ignored.
	a.newName = "renamed"
	a.newExe = `C:\new.exe`
	a.newArgs = "-new"
	if !a.saveService() {
		t.Fatalf("saveService: %s", a.dlgErr)
	}
	if _, err := os.Stat(filepath.Join(userData, "services", "demo")); err != nil {
		t.Errorf("the service directory went away: %v", err)
	}
	if _, err := os.Stat(filepath.Join(userData, "services", "renamed")); !os.IsNotExist(err) {
		t.Errorf("a renamed directory appeared: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(userData, "services", "demo", "demo.xml"))
	if err != nil {
		t.Fatalf("reading the configuration: %v", err)
	}
	for _, want := range []string{
		"<id>demo</id>",
		`<executable>C:\new.exe</executable>`,
		"<arguments>-new</arguments>",
	} {
		if !strings.Contains(string(data), want) {
			t.Errorf("configuration %q does not contain %q", data, want)
		}
	}
}

func TestDeleteService(t *testing.T) {
	userData := t.TempDir()
	mygo.App.SetPath(mygo.PathUserData, userData)
	t.Cleanup(func() { mygo.App.SetPath(mygo.PathUserData, "") })

	a := &app{}
	a.load()
	a.newName = "demo"
	a.newExe = `C:\Tools\demo.exe`
	if !a.addService() {
		t.Fatalf("addService: %s", a.dlgErr)
	}
	dir := filepath.Join(userData, "services", "demo")
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("the service was not created: %v", err)
	}

	a.deleting = "demo"
	a.deleteService()
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("the service directory remains: %v", err)
	}
	for _, s := range a.services {
		if s.name == "demo" {
			t.Error("the deleted service remains in the list")
		}
	}
}

func TestDeleteServiceFromMenu(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(dir, "demo", "demo.xml")
	if err := os.WriteFile(config, []byte("<service><id>demo</id><executable>x.exe</executable></service>"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &app{
		dir:      dir,
		winsw:    "winsw.exe",
		services: []service{{name: "demo", status: statusNotInstalled}},
	}
	tt := ui.NewTester(a.view, 480, 360)
	if err := tt.Click(locales.More()); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem(locales.Delete()); err != nil {
		t.Fatal(err)
	}
	if !a.confirmDelete {
		t.Fatal("the delete alert did not open")
	}
	if err := tt.Click(locales.DeleteAction()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "demo")); !os.IsNotExist(err) {
		t.Errorf("the service directory remains: %v", err)
	}
	if a.confirmDelete {
		t.Error("the delete alert stayed open")
	}
}
