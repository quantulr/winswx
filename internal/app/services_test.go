package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func TestLoadServices(t *testing.T) {
	userData := t.TempDir()
	mygo.App.SetPath(mygo.PathUserData, userData)
	t.Cleanup(func() { mygo.App.SetPath(mygo.PathUserData, "") })

	a := &App{}
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

func TestAddService(t *testing.T) {
	userData := t.TempDir()
	mygo.App.SetPath(mygo.PathUserData, userData)
	t.Cleanup(func() { mygo.App.SetPath(mygo.PathUserData, "") })

	a := &App{}
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

	a := &App{}
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

	a := &App{}
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

	a := &App{}
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

func TestAvatarColor(t *testing.T) {
	seen := map[ui.Color]bool{}
	for i := range avatarColors {
		c := avatarColor(i)
		if seen[c] {
			t.Errorf("color %v repeats at index %d", c, i)
		}
		seen[c] = true
	}
	if avatarColor(len(avatarColors)) != avatarColor(0) {
		t.Error("the colors do not cycle")
	}
}

func TestServiceWinsw(t *testing.T) {
	userData := t.TempDir()
	mygo.App.SetPath(mygo.PathUserData, userData)
	t.Cleanup(func() { mygo.App.SetPath(mygo.PathUserData, "") })

	bundled := filepath.Join(t.TempDir(), "winsw.exe")
	if err := os.WriteFile(bundled, []byte("wrapper"), 0o755); err != nil {
		t.Fatal(err)
	}
	a := &App{winsw: bundled}
	got, err := a.serviceWinsw()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(userData, "bin", "winsw.exe")
	if got != want {
		t.Fatalf("serviceWinsw = %q, want %q", got, want)
	}
	if data, err := os.ReadFile(got); err != nil || string(data) != "wrapper" {
		t.Fatalf("copied wrapper = %q, %v", data, err)
	}

	// Once copied, the bundled file may go: services keep their wrapper.
	if err := os.Remove(bundled); err != nil {
		t.Fatal(err)
	}
	if again, err := a.serviceWinsw(); err != nil || again != got {
		t.Fatalf("serviceWinsw again = %q, %v", again, err)
	}
}
