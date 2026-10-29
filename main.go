//go:generate go run ./scripts/fetch-winsw

package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/plugins/updater/native"
	"github.com/egoist/mygo/ui"

	"mygo/locales"
)

// serviceStatus is how Windows reports the state of a service.
type serviceStatus string

// The states a service shows.
const (
	statusChecking     serviceStatus = "Checking…"
	statusRunning      serviceStatus = "Running"
	statusStopped      serviceStatus = "Stopped"
	statusStarting     serviceStatus = "Starting"
	statusStopping     serviceStatus = "Stopping"
	statusPaused       serviceStatus = "Paused"
	statusNotInstalled serviceStatus = "Not installed"
	statusUnknown      serviceStatus = "Unknown"
)

// refreshIcon is a clockwise refresh arrow, drawn in the text color of
// the button that shows it.
var refreshIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12a9 9 0 1 1-9-9c2.52 0 4.93 1 6.74 2.74L21 8"/><path d="M21 3v5h-5"/></svg>`))

// plusIcon is a plus, drawn in the text color of the button that shows it.
var plusIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 5v14M5 12h14"/></svg>`))

// languageIcon is a globe, drawn in the text color of the button that
// shows it.
var languageIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3a15 15 0 0 1 0 18M12 3a15 15 0 0 0 0 18"/></svg>`))

// The actions of a card, drawn in the text color of their buttons.
var (
	playIcon    = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor"><path d="M9 6.2v11.6a1 1 0 0 0 1.52.85l9.3-5.8a1 1 0 0 0 0-1.7l-9.3-5.8A1 1 0 0 0 9 6.2z"/></svg>`))
	stopIcon    = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor"><rect x="6" y="6" width="12" height="12" rx="2.5"/></svg>`))
	installIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 3v12M7 10l5 5 5-5M5 21h14"/></svg>`))
	moreIcon    = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor"><circle cx="6" cy="12" r="1.7"/><circle cx="12" cy="12" r="1.7"/><circle cx="18" cy="12" r="1.7"/></svg>`))
)

// The rows of the More menu.
var (
	editIcon      = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 5l4 4M4 20l1-4.5L16.5 4a2.1 2.1 0 0 1 3 3L8 18.5z"/></svg>`))
	restartIcon   = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12a9 9 0 1 1-9-9c2.52 0 4.93 1 6.74 2.74L21 8"/><path d="M21 3v5h-5"/></svg>`))
	uninstallIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20v-9M8 15l4-4 4 4M5 4h14"/></svg>`))
	trashIcon     = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 7h16M10 4h4M6 7l1 13h10l1-13M10 11v6M14 11v6"/></svg>`))
	logsIcon      = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M6 3h9l4 4v14H6z"/><path d="M14 3v5h5M9 12h6M9 16h6"/></svg>`))
)

// avatarColors are the colors a service's badge picks from, by its name.
var avatarColors = []ui.Color{
	ui.Hex("#3D9BD1"), // zenith blue
	ui.Hex("#EE5F86"), // coral pink
	ui.Hex("#009B8B"), // vivid green
	ui.Hex("#CE8F24"), // orange yellow
	ui.Hex("#9455B0"), // indigo purple
}

// avatarColor picks the badge color of the service at index i, so that
// the services in view do not share one.
func avatarColor(i int) ui.Color {
	return avatarColors[i%len(avatarColors)]
}

// service is a directory in the services directory, and the status of the
// Windows service named after it.
type service struct {
	name   string
	status serviceStatus
}

// Cards keep their width as the window resizes: the grid adds or drops
// columns instead.
const (
	cardWidth  = 200
	cardHeight = 112
)

// app is the state the window shows. Its view builds the interface from
// it, on the main thread, whenever the window needs a frame: after input,
// after Window.Update, and while something animates.
type app struct {
	dir      string          // the services directory, in the user data directory
	services []service       // the directories in dir: a service each
	winsw    string          // the bundled WinSW executable, "" when it is missing
	busy     map[string]bool // the services with a WinSW command running
	opErr    string          // why the last operation failed
	err      error           // why the services could not be read, if they could not
	win      *mygo.Window    // the window, nil before it is made
	queryGen int             // counts status queries, so a stale one is ignored

	dialog        dialogMode // the dialog that shows, none when closed
	editDir       string     // the directory of the service being edited
	moreName      string     // the service whose More menu is open
	moreOpen      bool       // the More menu is built
	moreTarget    float32    // 1 shown, 0 closing
	moreFresh     bool       // seed the opening animation
	deleting      string     // the service asked about for deletion
	confirmDelete bool       // the delete alert shows
	logsDir       string     // the service whose logs the dialog shows
	logsOptions   []string   // the log files the segmented control offers
	logsExisting  []bool     // whether each option's file exists
	logsFile      string     // the log file shown
	logsText      string     // its tail
	logsTrunc     bool       // the tail is not the whole file
	logsErr       string     // why the logs could not be read
	logsTab       int        // the log chosen in the segmented control
	logsScroll    ui.ScrollState
	dlgOpen       bool    // DialogBase's open flag
	dlgTarget     float32 // 1 shown, 0 closing
	dlgFresh      bool    // seed the opening animation
	newName       string  // the name in the dialog
	newExe        string  // the executable in the dialog
	newArgs       string  // the arguments in the dialog
	dlgErr        string  // why the dialog could not be saved
}

// dialogMode names the form the dialog shows.
type dialogMode uint8

// The forms of the dialog.
const (
	dialogNone dialogMode = iota
	dialogAdd
	dialogEdit
	dialogLogs
)

// winswExe returns the path of the WinSW executable that ships with the
// app in its resources.
func winswExe() (string, error) {
	dir, err := mygo.App.Path(mygo.PathResources)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "bin", "winsw.exe"), nil
}

// parseStatus maps what WinSW prints for a service to a serviceStatus:
// v3 prints systemd-style states, such as "Active (running)".
func parseStatus(out string) serviceStatus {
	switch strings.TrimSpace(out) {
	case "Active (running)":
		return statusRunning
	case "Active (starting)":
		return statusStarting
	case "Active (stopping)":
		return statusStopping
	case "Active (pausing)", "Active (paused)", "Active (continuing)":
		return statusPaused
	case "Inactive (stopped)":
		return statusStopped
	case "NonExistent":
		return statusNotInstalled
	}
	return statusUnknown
}

// queryService asks WinSW for the state of the service that config
// describes. WinSW prints the state to stdout even when it exits non-zero:
// v3.0.0 exits 1 for a running service, so the output is what counts.
func queryService(winsw, config string) serviceStatus {
	if winsw == "" || config == "" {
		return statusUnknown
	}
	cmd := exec.Command(winsw, "status", config)
	hideWindow(cmd)
	out, err := cmd.Output()
	if status := parseStatus(string(out)); status != statusUnknown {
		return status
	}
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok && exit.ExitCode() == 1060 {
			return statusNotInstalled
		}
	}
	return statusUnknown
}

// serviceConfig returns the WinSW configuration of the service in the
// directory dir/name: the XML named after it, or the only XML in it.
func serviceConfig(dir, name string) string {
	serviceDir := filepath.Join(dir, name)
	path := filepath.Join(serviceDir, name+".xml")
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		return path
	}
	entries, err := os.ReadDir(serviceDir)
	if err != nil {
		return ""
	}
	found := ""
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".xml") {
			continue
		}
		if found != "" {
			// More than one XML: not clear which one describes the service.
			return ""
		}
		found = filepath.Join(serviceDir, entry.Name())
	}
	return found
}

// serviceXML returns the WinSW configuration of a service.
func serviceXML(name, executable, arguments string) ([]byte, error) {
	cfg := struct {
		XMLName    xml.Name `xml:"service"`
		ID         string   `xml:"id"`
		Executable string   `xml:"executable"`
		Arguments  string   `xml:"arguments,omitempty"`
	}{ID: name, Executable: executable, Arguments: arguments}
	body, err := xml.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, err
	}
	out := append([]byte(xml.Header), body...)
	return append(out, '\n'), nil
}

// validateServiceName reports why name cannot be the name of a service
// directory, or "" when it can.
func validateServiceName(name string) string {
	switch {
	case name == "":
		return locales.ErrNeedsName()
	case name == "." || name == "..":
		return locales.ErrNameDots()
	case strings.TrimSpace(name) != name:
		return locales.ErrNameEndsInSpace()
	case strings.ContainsAny(name, `\/:*?"<>|`):
		return locales.ErrNameChars()
	}
	for _, r := range name {
		if r < 32 {
			return locales.ErrNameControl()
		}
	}
	base := name
	if i := strings.IndexByte(base, '.'); i >= 0 {
		base = base[:i]
	}
	switch strings.ToUpper(base) {
	case "CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return locales.ErrNameReserved()
	}
	return ""
}

// addService creates a service directory and its WinSW configuration from
// the fields of the Add Service dialog.
func (a *app) addService() bool {
	name := a.newName // not trimmed: a name with spaces at its ends is refused
	exe := strings.TrimSpace(a.newExe)
	args := strings.TrimSpace(a.newArgs)
	if msg := validateServiceName(name); msg != "" {
		a.dlgErr = msg
		return false
	}
	if exe == "" {
		a.dlgErr = locales.ErrExecutable()
		return false
	}
	if a.dir == "" {
		a.dlgErr = locales.ErrServicesDir()
		return false
	}
	dir := filepath.Join(a.dir, name)
	if _, err := os.Stat(dir); err == nil {
		a.dlgErr = locales.ServiceExists(name)
		return false
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		a.dlgErr = err.Error()
		return false
	}
	config := filepath.Join(dir, name+".xml")
	body, err := serviceXML(name, exe, args)
	if err == nil {
		err = os.WriteFile(config, body, 0o644)
	}
	if err != nil {
		_ = os.RemoveAll(dir) // do not leave half a service behind
		a.dlgErr = err.Error()
		return false
	}
	a.dlgErr = ""
	a.newName, a.newExe, a.newArgs = "", "", ""
	a.load()
	return true
}

// statusLabel says a service's status in the app's language.
func statusLabel(s serviceStatus) string {
	switch s {
	case statusChecking:
		return locales.StatusChecking()
	case statusRunning:
		return locales.StatusRunning()
	case statusStopped:
		return locales.StatusStopped()
	case statusStarting:
		return locales.StatusStarting()
	case statusStopping:
		return locales.StatusStopping()
	case statusPaused:
		return locales.StatusPaused()
	case statusNotInstalled:
		return locales.StatusNotInstalled()
	}
	return locales.StatusUnknown()
}

// statusColor is the color that shows status: green running, grey
// stopped, amber in between, red unknown.
func statusColor(t *ui.Theme, s serviceStatus) ui.Color {
	switch s {
	case statusRunning:
		return t.Success
	case statusStopped, statusChecking:
		return t.TextMuted
	case statusStarting, statusStopping, statusPaused, statusNotInstalled:
		return t.Warning
	default:
		return t.Danger
	}
}

// load creates the services directory in the user data directory if it is
// missing, then lists the directories in it. Each directory is a service.
// The statuses are queried in the background.
func (a *app) load() {
	userData, err := mygo.App.Path(mygo.PathUserData)
	if err == nil {
		a.dir = filepath.Join(userData, "services")
		err = os.MkdirAll(a.dir, 0o755)
	}
	var entries []os.DirEntry
	if err == nil {
		entries, err = os.ReadDir(a.dir)
	}
	if err != nil {
		a.err = err
		a.services = nil
		return
	}
	a.err = nil
	a.winsw = ""
	if exe, err := winswExe(); err == nil {
		if info, err := os.Stat(exe); err == nil && !info.IsDir() {
			a.winsw = exe
		}
	}
	known := make(map[string]serviceStatus, len(a.services))
	for _, s := range a.services {
		known[s.name] = s.status
	}
	a.services = a.services[:0]
	for _, entry := range entries {
		if entry.IsDir() {
			name := entry.Name()
			status, ok := known[name]
			if !ok {
				status = statusUnknown
				if a.winsw != "" && serviceConfig(a.dir, name) != "" {
					status = statusChecking
				}
			}
			a.services = append(a.services, service{name: name, status: status})
		}
	}
	a.refreshStatuses()
}

// refreshStatuses asks WinSW for the state of every service in the
// background: a query takes hundreds of milliseconds, and the view must
// not wait for them.
func (a *app) refreshStatuses() {
	if a.win == nil || a.winsw == "" {
		return
	}
	a.queryGen++
	gen := a.queryGen
	winsw, win := a.winsw, a.win
	names := make([]string, len(a.services))
	configs := make([]string, len(a.services))
	for i, s := range a.services {
		names[i] = s.name
		configs[i] = serviceConfig(a.dir, s.name)
	}
	go func() {
		statuses := make([]serviceStatus, len(names))
		sem := make(chan struct{}, 4)
		var wg sync.WaitGroup
		for i := range names {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				statuses[i] = queryService(winsw, configs[i])
			}(i)
		}
		wg.Wait()
		byName := make(map[string]serviceStatus, len(names))
		for i, name := range names {
			byName[name] = statuses[i]
		}
		win.Update(func() {
			if gen != a.queryGen {
				return // a newer query supersedes this one
			}
			for i := range a.services {
				if s, ok := byName[a.services[i].name]; ok {
					a.services[i].status = s
				}
			}
		})
	}()
}

// run executes a WinSW command on a service in the background, then
// reloads the services to show the result. WinSW installs and starts
// services with the rights of the app, which needs to be elevated.
// serviceWinsw returns the WinSW executable services are installed and
// run from. It lives in the user data directory, apart from the app's own
// files, so that a running service does not lock what an update replaces.
func (a *app) serviceWinsw() (string, error) {
	userData, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return "", err
	}
	dest := filepath.Join(userData, "bin", "winsw.exe")
	if _, err := os.Stat(dest); err == nil {
		return dest, nil
	}
	if a.winsw == "" {
		return "", errors.New("the bundled winsw.exe is missing")
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", err
	}
	src, err := os.Open(a.winsw)
	if err != nil {
		return "", err
	}
	defer src.Close()
	tmp := dest + ".download"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(out, src); err != nil {
		_ = out.Close()
		_ = os.Remove(tmp)
		return "", err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	return dest, nil
}

func (a *app) run(name, command string) {
	if a.win == nil || a.winsw == "" || a.busy[name] {
		return
	}
	config := serviceConfig(a.dir, name)
	if config == "" {
		a.opErr = name + ": " + locales.ErrNoConfig()
		return
	}
	winsw, err := a.serviceWinsw()
	if err != nil {
		a.opErr = err.Error()
		return
	}
	if a.busy == nil {
		a.busy = make(map[string]bool)
	}
	a.busy[name] = true
	a.opErr = ""
	win := a.win
	win.Update(func() {}) // redraw with the service busy
	go func() {
		cmd := exec.Command(winsw, command, config)
		hideWindow(cmd)
		out, err := cmd.CombinedOutput()
		win.Update(func() {
			delete(a.busy, name)
			if err != nil {
				a.opErr = name + " " + command + ": " + err.Error()
				if line := firstLine(string(out)); line != "" {
					a.opErr += ": " + line
				}
			}
			a.load()
		})
	}()
}

// firstLine returns the first non-empty line of s.
func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			return line
		}
	}
	return ""
}

// petalWash returns kero's petal backdrop: a wash of lavender, rose and
// cream in the light, and of deep plum in the dark, lit from the top
// right.
func petalWash(dark bool) (base, glow ui.LinearGradient) {
	stops := func(hexes ...string) []ui.GradientStop {
		out := make([]ui.GradientStop, len(hexes))
		for i, h := range hexes {
			out[i] = ui.GradientStop{Color: ui.Hex(h), At: float32(i) / float32(len(hexes)-1)}
		}
		return out
	}
	glowOf := func(c ui.Color) ui.LinearGradient {
		return ui.LinearGradient{Angle: 225, Oklab: true, Stops: []ui.GradientStop{
			{Color: c, At: 0},
			{Color: c.Alpha(0), At: 0.5},
		}}
	}
	if dark {
		return ui.LinearGradient{Angle: 180, Oklab: true,
				Stops: stops("#17141f", "#191620", "#1b1916")},
			glowOf(ui.RGBA(120, 70, 120, 0.22))
	}
	return ui.LinearGradient{Angle: 180, Oklab: true,
			Stops: stops("#ebe0f3", "#f3e7f1", "#f7eee4", "#f6efcf")},
		glowOf(ui.RGBA(247, 220, 241, 0.75))
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	// kero's petal wash, which the glass of the Refresh button bends and
	// frosts.
	base, glow := petalWash(t.Dark)
	ui.Box(c).Fill().LinearGradient(base).Children(func() {
		ui.Column(c).Fill().LinearGradient(glow).Children(func() {
			// The view draws the title bar: a strip of glass under the
			// window controls that drags the window.
			bar := c.TitleBar()
			ui.Row(c).Height(max(bar.Height, 32)).
				Padding(0, bar.Right+20, 0, bar.Left+20).
				AlignItems(ui.Center).DragWindow().Children(func() {
				ui.Text(c, "winswx").FontSize(12).TextColor(t.TextMuted)
			})
			ui.Column(c).Grow(1).Gap(12).Padding(16, 20).Children(func() {
				ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Text(c, locales.Services()).FontSize(20).Bold()
					ui.Textf(c, "%d", len(a.services)).TextColor(t.TextMuted)
					ui.Spacer(c).Grow(1)
					add := ui.Button(c, "").Label(locales.AddService()).Tooltip(locales.AddService()).
						Padding(6, 8).BorderWidth(0).Radius(10).Cursor(ui.CursorPointer).
						Material(glass.Glass{Interactive: true}).Children(func() {
						ui.Icon(c, plusIcon).FontSize(16)
					})
					if add.Clicked() {
						a.openAddDialog()
					}
					refresh := ui.Button(c, "").Label(locales.Refresh()).Tooltip(locales.Refresh()).
						Padding(6, 8).BorderWidth(0).Radius(10).Cursor(ui.CursorPointer).
						Material(glass.Glass{Interactive: true}).Children(func() {
						ui.Icon(c, refreshIcon).FontSize(16)
					})
					if refresh.Clicked() {
						a.load()
					}
					language := ui.Button(c, "").Label(locales.Language()).Tooltip(locales.Language()).
						Padding(6, 8).BorderWidth(0).Radius(10).Cursor(ui.CursorPointer).
						Material(glass.Glass{Interactive: true}).Children(func() {
						ui.Icon(c, languageIcon).FontSize(16)
					})
					language.Menu(func(m *ui.Menu) {
						for _, tag := range mygo.App.Languages() {
							if m.Item(i18n.Autonym(tag)).Checked(tag == mygo.App.Locale()).Chosen() {
								mygo.App.SetLocale(tag)
							}
						}
					})
				})
				if a.opErr != "" {
					ui.Text(c, a.opErr).FontSize(12).TextColor(t.Danger).Wrap()
				}
				switch {
				case a.err != nil:
					ui.Text(c, a.err.Error()).TextColor(t.Danger)
				case len(a.services) == 0:
					ui.Column(c).Grow(1).Center().Children(func() {
						ui.Text(c, locales.NoServices()).TextColor(t.TextMuted)
					})
				default:
					gap := t.Space(2)
					winW, _ := c.Size()
					avail := winW - 2*20 // the column's horizontal padding
					cols := max(1, int((avail+gap)/(cardWidth+gap)))
					tracks := make([]ui.Track, cols)
					for i := range tracks {
						tracks[i] = ui.Fixed(cardWidth)
					}
					ui.Scroll(c).Grow(1).Children(func() {
						ui.Grid(c).FillWidth().ColumnTracks(tracks...).Gap(gap).
							Justify(ui.Start).Children(func() {
							for i, s := range a.services {
								a.serviceCard(c, i, s)
							}
						})
					})
				}
				ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Spacer(c).Grow(1)
					if a.winsw == "" {
						ui.Text(c, locales.WinswMissing()).TextColor(t.Danger).FontSize(12)
					}
				})
			})
		})
	})
	a.dialogView(c)
	a.deleteAlert(c)
}

// dialogView shows the Add or Edit Service form, easing it in and out.
func (a *app) dialogView(c *ui.Context) {
	if a.dialog == dialogNone {
		return
	}
	a.dlgOpen = true
	t := c.Theme()
	winW, winH := c.Size()
	w := min(360, max(160, winW-80)) // the panel keeps 20 DIP of padding aside
	ui.DialogBase(c, &a.dlgOpen, func(backdrop, panel ui.Element) {
		anim := ui.Animation{Duration: 160 * time.Millisecond, Ease: ui.EaseOut}
		target := a.dlgTarget
		p := panel.AnimateTo("in", target, anim)
		dim := backdrop.AnimateTo("dim", target, anim)
		if a.dlgFresh {
			// An animated value starts at its first target: show the
			// dialog at zero for a frame, then ease it in on the next.
			a.dlgFresh = false
			a.dlgTarget = 1
			c.Invalidate()
		}
		p = min(1, max(0, p))
		backdrop.Background(ui.RGBA(0, 0, 0, 0.4*dim))
		panel.Padding(t.Space(5)).Gap(t.Space(3)).Radius(t.Space(2.5)).
			Background(t.Background).
			MaxWidth(winW-t.Space(10)).MaxHeight(winH-t.Space(10)).
			Shadow(0, 10*p, 30*p, 0, ui.RGBA(0, 0, 0, 0.3*p)).
			Scale(0.96+0.04*p, 0.96+0.04*p).Opacity(p)
		if a.dialog == dialogLogs {
			lw := min(800, max(320, winW-80))
			a.logsForm(c, lw, winH)
		} else {
			title, action := locales.AddService(), locales.Add()
			if a.dialog == dialogEdit {
				title, action = locales.EditService(), locales.Save()
			}
			ui.Text(c, title).FontSize(18).Bold()
			name := ui.TextInput(c, &a.newName).Placeholder(locales.ServiceName()).
				Label("Name").AutoFocus().Width(w)
			if a.dialog == dialogEdit {
				name.Disabled(true).Tooltip(locales.ServiceIDReadOnly())
			}
			var exe ui.Element
			ui.Row(c).Gap(8).Width(w).Children(func() {
				exe = ui.TextInput(c, &a.newExe).Placeholder(`C:\path\to\service.exe`).
					Label("Executable").Grow(1)
				if ui.Button(c, locales.Browse()).Cursor(ui.CursorPointer).Clicked() {
					a.pickExecutable()
				}
			})
			args := ui.TextInput(c, &a.newArgs).Placeholder(locales.ArgumentsOptional()).
				Label("Arguments").Width(w)
			if a.dlgErr != "" {
				ui.Text(c, a.dlgErr).FontSize(12).TextColor(t.Danger).Wrap().Width(w)
			}
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c, locales.Cancel()).Clicked() {
					a.closeDialog(c)
				}
				if ui.PrimaryButton(c, action).Clicked() ||
					name.Submitted() || exe.Submitted() || args.Submitted() {
					saved := false
					if a.dialog == dialogEdit {
						saved = a.saveService()
					} else {
						saved = a.addService()
					}
					if saved {
						a.closeDialog(c)
					}
				}
			})
		}
		// Faded away: take the dialog out of the tree.
		if a.dlgTarget == 0 && p < 0.01 {
			a.dialog = dialogNone
			a.dlgOpen = false
		}
	})
	if !a.dlgOpen {
		// A click outside or Escape closed it: ease it out.
		a.closeDialog(c)
	}
}

// openAddDialog shows the form of a new service.
func (a *app) openAddDialog() {
	a.dialog = dialogAdd
	a.dlgOpen = true
	a.dlgTarget = 0
	a.dlgFresh = true
	a.dlgErr = ""
	a.newName, a.newExe, a.newArgs = "", "", ""
}

// openEditDialog shows the form of an existing service, filled in from
// its WinSW configuration.
func (a *app) openEditDialog(name string) {
	config := serviceConfig(a.dir, name)
	if config == "" {
		a.opErr = name + ": " + locales.ErrNoConfig()
		return
	}
	data, err := os.ReadFile(config)
	if err != nil {
		a.opErr = err.Error()
		return
	}
	var cfg struct {
		Executable string `xml:"executable"`
		Arguments  string `xml:"arguments"`
	}
	if err := xml.Unmarshal(data, &cfg); err != nil {
		a.opErr = name + ": " + err.Error()
		return
	}
	a.dialog = dialogEdit
	a.editDir = name
	a.dlgOpen = true
	a.dlgTarget = 0
	a.dlgFresh = true
	a.dlgErr = ""
	a.newName = name
	a.newExe = cfg.Executable
	a.newArgs = cfg.Arguments
}

// closeDialog keeps the dialog built while it eases out.
func (a *app) closeDialog(c *ui.Context) {
	if a.dlgTarget == 0 {
		return
	}
	a.dlgOpen = true
	a.dlgTarget = 0
	c.Invalidate()
}

// openLogsDialog shows the log files of a service.
func (a *app) openLogsDialog(name string) {
	a.logsDir = filepath.Join(a.dir, name)
	a.logsOptions = nil
	a.logsExisting = nil
	a.logsFile = ""
	a.logsTab = 0
	a.logsText, a.logsTrunc, a.logsErr = "", false, ""
	a.dialog = dialogLogs
	a.dlgOpen = true
	a.dlgTarget = 0
	a.dlgFresh = true
	a.dlgErr = ""
	a.reloadLogs()
}

// reloadLogs lists the log files in the service's directory and reads the
// tail of the selected one.
func (a *app) reloadLogs() {
	a.logsOptions = a.logsOptions[:0]
	a.logsExisting = a.logsExisting[:0]
	a.logsText, a.logsTrunc, a.logsErr = "", false, ""
	entries, err := os.ReadDir(a.logsDir)
	if err != nil {
		a.logsErr = err.Error()
		return
	}
	base := filepath.Base(a.logsDir)
	standard := []string{base + ".wrapper.log", base + ".out.log", base + ".err.log"}
	existing := make(map[string]bool, len(entries))
	var extra []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".log") {
			continue
		}
		name := entry.Name()
		existing[name] = true
		if name != standard[0] && name != standard[1] && name != standard[2] {
			extra = append(extra, name)
		}
	}
	a.logsOptions = append(a.logsOptions, standard...)
	a.logsOptions = append(a.logsOptions, extra...)
	for _, name := range a.logsOptions {
		a.logsExisting = append(a.logsExisting, existing[name])
	}

	// Keep the chosen log while it exists; else show the wrapper log,
	// else the first that exists.
	a.logsTab = 0
	kept := false
	for i, name := range a.logsOptions {
		if name == a.logsFile && a.logsExisting[i] {
			a.logsTab, kept = i, true
			break
		}
	}
	if !kept {
		a.logsFile = ""
		for i, name := range a.logsOptions {
			if a.logsExisting[i] && strings.HasSuffix(name, ".wrapper.log") {
				a.logsFile, a.logsTab = name, i
				break
			}
		}
		if a.logsFile == "" {
			for i, name := range a.logsOptions {
				if a.logsExisting[i] {
					a.logsFile, a.logsTab = name, i
					break
				}
			}
		}
	}
	if a.logsFile == "" {
		return
	}
	text, trunc, err := readLogTail(filepath.Join(a.logsDir, a.logsFile), 64<<10)
	if err != nil {
		a.logsErr = err.Error()
		return
	}
	a.logsText, a.logsTrunc = text, trunc
	a.logsScroll.X, a.logsScroll.Y = 0, math.MaxFloat32 // show the newest lines
}

// logLabel names a log file without its service's prefix.
func (a *app) logLabel(name string) string {
	base := filepath.Base(a.logsDir)
	if rest, ok := strings.CutPrefix(name, base+"."); ok {
		return rest
	}
	return name
}

// readLogTail returns the last size bytes of path, dropping the partial
// line at the cut, and reports whether anything was left out.
func readLogTail(path string, size int64) (string, bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", false, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return "", false, err
	}
	off := info.Size() - size
	if off < 0 {
		off = 0
	}
	if _, err := f.Seek(off, io.SeekStart); err != nil {
		return "", false, err
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return "", false, err
	}
	s := string(data)
	if off > 0 {
		if i := strings.IndexByte(s, '\n'); i >= 0 {
			s = s[i+1:]
		}
	}
	return strings.ToValidUTF8(s, "�"), off > 0, nil
}

// logsForm shows the log files of a service and the tail of the selected
// one, which Refresh reads again.
func (a *app) logsForm(c *ui.Context, w, winH float32) {
	t := c.Theme()
	ui.Row(c).AlignItems(ui.Center).Gap(8).Width(w).Children(func() {
		ui.Text(c, locales.Logs()).FontSize(18).Bold()
		ui.Text(c, filepath.Base(a.logsDir)).TextColor(t.TextMuted)
		ui.Spacer(c).Grow(1)
		if ui.Button(c, locales.Refresh()).FontSize(12).Padding(3, 7).
			Cursor(ui.CursorPointer).Clicked() {
			a.reloadLogs()
		}
	})
	hasLogs := false
	for _, exists := range a.logsExisting {
		if exists {
			hasLogs = true
			break
		}
	}
	if hasLogs {
		labels := make([]string, len(a.logsOptions))
		for i, name := range a.logsOptions {
			labels[i] = a.logLabel(name)
		}
		ui.Row(c).Width(w).Justify(ui.Center).Children(func() {
			edge := t.Text.Alpha(0.14)
			if t.Dark {
				edge = t.Text.Alpha(0.20)
			}
			seg := ui.SegmentedBase(c, &a.logsTab, len(labels))
			seg.Track.Padding(3).Radius(10).Border(1, edge).
				Shadow(0, 2, 10, 0, ui.RGBA(0, 0, 0, 0.16)).
				Label(locales.Logs()).Material(glass.Glass{}).
				Children(func() {
					for i, label := range labels {
						s := seg.Segment(i).Padding(4, 12).Radius(7).
							Label(label)
						if !a.logsExisting[i] {
							s.Disabled(true).TextColor(t.TextMuted.Alpha(0.55))
						} else {
							s.Cursor(ui.CursorPointer)
							if i == a.logsTab && a.logsOptions[i] == a.logsFile {
								s.Background(t.Accent).TextColor(t.AccentText)
							} else {
								s.TextColor(t.Text)
								if s.Hovered() {
									s.Background(t.Text.Alpha(0.07))
								}
							}
						}
						s.Children(func() { ui.Text(c, label).SingleLine() })
					}
				})
		})
		// The control writes the chosen index.
		if a.logsTab >= 0 && a.logsTab < len(a.logsOptions) {
			name := a.logsOptions[a.logsTab]
			if a.logsExisting[a.logsTab] && name != a.logsFile {
				a.logsFile = name
				a.reloadLogs()
			}
		}
	}
	// The log area and its note keep their room whatever the chosen file
	// holds, so that switching logs does not resize the dialog.
	height := max(80, winH-250)
	switch {
	case a.logsErr != "":
		ui.Text(c, a.logsErr).FontSize(12).TextColor(t.Danger).Wrap().Width(w)
	case a.logsFile == "":
		ui.Text(c, locales.NoLogs()).TextColor(t.TextMuted).Width(w)
	default:
		ui.ScrollBoth(c).Width(w).Height(height).Padding(8).Radius(8).
			TrackScroll(&a.logsScroll).
			Background(t.Surface).Border(1, t.Border).Children(func() {
			if a.logsText == "" {
				ui.Text(c, locales.LogsEmpty()).TextColor(t.TextMuted)
				return
			}
			ui.Text(c, a.logsText).Font("Consolas").FontSize(12).
				LineHeight(1.3).NoWrap().Selectable()
		})
		note := ""
		if a.logsTrunc {
			note = locales.LogsTruncated()
		}
		ui.Row(c).Height(16).Width(w).Children(func() {
			if note != "" {
				ui.Text(c, note).FontSize(12).TextColor(t.TextMuted)
			}
		})
	}
	ui.Row(c).Gap(8).Justify(ui.End).Width(w).Children(func() {
		if ui.Button(c, locales.Close()).Clicked() {
			a.closeDialog(c)
		}
	})
}

// deleteAlert asks before removing a service and its files.
func (a *app) deleteAlert(c *ui.Context) {
	switch ui.AlertDialog(c, &a.confirmDelete, locales.DeleteTitle(a.deleting),
		locales.DeleteMessage(), locales.Cancel(), locales.DeleteAction()) {
	case 1:
		a.deleteService()
	}
}

// deleteService removes the directory of the service being deleted.
func (a *app) deleteService() {
	name := a.deleting
	a.deleting = ""
	if name == "" || a.dir == "" || name == "." || name == ".." ||
		strings.ContainsAny(name, `/\`) {
		return
	}
	if err := os.RemoveAll(filepath.Join(a.dir, name)); err != nil {
		a.opErr = err.Error()
		return
	}
	a.opErr = ""
	a.load()
}

// saveService applies the dialog's fields to the service being edited.
// Its ID never changes: the fields that shadow it are ignored.
func (a *app) saveService() bool {
	name := a.editDir
	exe := strings.TrimSpace(a.newExe)
	args := strings.TrimSpace(a.newArgs)
	if exe == "" {
		a.dlgErr = locales.ErrExecutable()
		return false
	}
	config := serviceConfig(a.dir, name)
	if config == "" {
		a.dlgErr = locales.ErrNoConfig()
		return false
	}
	data, err := os.ReadFile(config)
	if err != nil {
		a.dlgErr = err.Error()
		return false
	}
	updated, err := updateServiceXML(data, name, exe, args)
	if err != nil {
		a.dlgErr = err.Error()
		return false
	}
	if err := os.WriteFile(config, updated, 0o644); err != nil {
		a.dlgErr = err.Error()
		return false
	}
	a.dlgErr = ""
	a.newName, a.newExe, a.newArgs = "", "", ""
	a.load()
	return true
}

// updateServiceXML returns WinSW configuration data with the id,
// executable and arguments replaced, keeping the other elements as they
// are.
func updateServiceXML(data []byte, id, executable, arguments string) ([]byte, error) {
	values := map[string]string{"id": id, "executable": executable, "arguments": arguments}
	dec := xml.NewDecoder(bytes.NewReader(data))
	var out bytes.Buffer
	out.WriteString(xml.Header)
	enc := xml.NewEncoder(&out)
	found := make(map[string]bool, len(values))
	depth := 0
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.ProcInst:
			if strings.EqualFold(t.Target, "xml") {
				continue // the header is already written
			}
		case xml.StartElement:
			if value, ok := values[t.Name.Local]; ok && depth == 1 {
				found[t.Name.Local] = true
				if t.Name.Local == "arguments" && arguments == "" {
					if err := dec.Skip(); err != nil {
						return nil, err
					}
					continue
				}
				if err := enc.EncodeToken(t); err != nil {
					return nil, err
				}
				if err := enc.EncodeToken(xml.CharData(value)); err != nil {
					return nil, err
				}
				if err := enc.EncodeToken(t.End()); err != nil {
					return nil, err
				}
				if err := dec.Skip(); err != nil {
					return nil, err
				}
				continue
			}
			depth++
		case xml.EndElement:
			depth--
			if depth == 0 {
				for _, name := range []string{"id", "executable", "arguments"} {
					if found[name] || (name == "arguments" && arguments == "") {
						continue
					}
					start := xml.StartElement{Name: xml.Name{Local: name}}
					_ = enc.EncodeToken(xml.CharData("\n  "))
					_ = enc.EncodeToken(start)
					_ = enc.EncodeToken(xml.CharData(values[name]))
					_ = enc.EncodeToken(start.End())
				}
			}
		}
		if err := enc.EncodeToken(tok); err != nil {
			return nil, err
		}
	}
	if err := enc.Flush(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// pickExecutable asks for the executable of the service with the system's
// file dialog.
func (a *app) pickExecutable() {
	if a.win == nil {
		return
	}
	win := a.win
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent:      win,
			Title:       "Choose an executable",
			ButtonLabel: "Choose",
			Filters:     []mygo.FileFilter{{Name: "Programs", Extensions: []string{"exe"}}},
		})
		win.Update(func() {
			switch {
			case err != nil:
				a.dlgErr = err.Error()
			case len(paths) > 0:
				a.newExe = paths[0]
				a.dlgErr = ""
			}
		})
	}()
}

// serviceCard draws a service as a card in the grid, with its first
// letter as a badge, its status under its name, and its actions below.
func (a *app) serviceCard(c *ui.Context, i int, s service) {
	t := c.Theme()
	busy := a.busy[s.name]
	status := s.status
	label := statusLabel(status)
	color := statusColor(t, status)
	if busy {
		label, color = locales.Working(), t.Accent
	}
	card := ui.Column(c).Width(cardWidth).Height(cardHeight).Label(s.name).
		Gap(10).Padding(14).Radius(12).Justify(ui.SpaceBetween)
	// The shadow deepens and a faint tint fades in as the pointer comes
	// over the card, and both ease back as it leaves.
	lift := float32(0)
	tint := ui.Transparent
	if card.Hovered() {
		lift, tint = 1, t.Text.Alpha(0.06)
	}
	anim := ui.Animation{Duration: 160 * time.Millisecond, Ease: ui.EaseOut}
	lift = card.AnimateTo("lift", lift, anim)
	tint = card.AnimateColor("tint", tint, anim)
	base, hovered := float32(0.07), float32(0.16)
	if t.Dark {
		base, hovered = 0.28, 0.48
	}
	card.Material(glass.Glass{Tint: tint})
	card.Shadow(0, 2+6*lift, 8+16*lift, 0,
		ui.RGBA(0, 0, 0, base+(hovered-base)*lift))
	card.Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
			initial := strings.ToUpper(string([]rune(s.name)[0]))
			tint := avatarColor(i)
			ui.Box(c).Width(34).Height(34).Radius(9).Center().
				Background(tint.Alpha(0.16)).Children(func() {
				ui.Text(c, initial).FontSize(16).Bold().TextColor(tint)
			})
			ui.Column(c).Gap(2).Children(func() {
				ui.Text(c, s.name).Bold().SingleLine()
				ui.Row(c).AlignItems(ui.Center).Gap(5).Children(func() {
					ui.Box(c).Width(7).Height(7).Radius(4).Background(color)
					ui.Text(c, label).FontSize(12).TextColor(color).SingleLine()
				})
			})
		})
		a.serviceButtons(c, s, busy)
	})
}

// serviceButtons shows the actions of a service, by its status.
// actionButton is a button with an icon before its label.
func actionButton(c *ui.Context, primary bool, icon *ui.SVG, label string) ui.Element {
	var b ui.Element
	if primary {
		b = ui.PrimaryButton(c, "")
	} else {
		b = ui.Button(c, "")
	}
	return b.Label(label).Gap(4).Children(func() {
		ui.Icon(c, icon).FontSize(14)
		ui.Text(c, label).SingleLine()
	})
}

func (a *app) serviceButtons(c *ui.Context, s service, busy bool) {
	installed := s.status == statusRunning || s.status == statusStopped ||
		s.status == statusStarting || s.status == statusStopping || s.status == statusPaused
	ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
		label, command := "", ""
		var icon *ui.SVG
		switch s.status {
		case statusRunning, statusPaused, statusStarting:
			label, command, icon = locales.Stop(), "stop", stopIcon
		case statusStopped:
			label, command, icon = locales.Start(), "start", playIcon
		case statusNotInstalled:
			label, command, icon = locales.Install(), "install", installIcon
		}
		if label != "" {
			b := actionButton(c, command != "stop", icon, label)
			if b.Padding(3, 7).Cursor(ui.CursorPointer).Disabled(busy).Clicked() {
				a.run(s.name, command)
			}
		}
		editable := serviceConfig(a.dir, s.name) != ""
		more := actionButton(c, false, moreIcon, locales.More()).
			Padding(3, 7).Cursor(ui.CursorPointer).Disabled(busy)
		if more.Clicked() {
			if a.moreName == s.name && a.moreOpen {
				a.closeMore(c)
			} else {
				a.moreName, a.moreOpen = s.name, true
				a.moreTarget, a.moreFresh = 0, true
			}
		}
		if a.moreName == s.name && a.moreOpen {
			finished := false
			ui.PopoverBase(c, more, &a.moreOpen, func(panel ui.Element) {
				t := c.Theme()
				anim := ui.Animation{Duration: 140 * time.Millisecond, Ease: ui.EaseOut}
				p := panel.AnimateTo("in", a.moreTarget, anim)
				if a.moreFresh {
					a.moreFresh = false
					a.moreTarget = 1
					c.Invalidate()
				}
				p = min(1, max(0, p))
				shade, edge := float32(0.35), float32(0.12)
				if t.Dark {
					shade, edge = 0.60, 0.18
				}
				panel.Padding(6).Radius(14).MinWidth(170).
					Opacity(p).
					Scale(0.97+0.03*p, 0.97+0.03*p).
					Shadow(0, 8*p, 20*p, 0, ui.RGBA(0, 0, 0, shade*p)).
					Border(1, t.Text.Alpha(edge*p)).
					Material(glass.Glass{})
				if menuItem(c, editIcon, locales.Edit(), !busy && editable) {
					a.closeMore(c)
					a.openEditDialog(s.name)
				}
				if menuItem(c, logsIcon, locales.ViewLogs(), !busy) {
					a.closeMore(c)
					a.openLogsDialog(s.name)
				}
				menuSeparator(c)
				if menuItem(c, restartIcon, locales.Restart(), !busy && installed) {
					a.closeMore(c)
					a.run(s.name, "restart")
				}
				if menuItem(c, uninstallIcon, locales.Uninstall(), !busy && s.status == statusStopped) {
					a.closeMore(c)
					a.run(s.name, "uninstall")
				}
				menuSeparator(c)
				if menuItem(c, trashIcon, locales.Delete(), !busy && !installed) {
					a.closeMore(c)
					a.deleting = s.name
					a.confirmDelete = true
				}
				if a.moreTarget == 0 && p < 0.01 {
					// Faded away: take it out of the tree.
					a.moreOpen = false
					a.moreName = ""
					finished = true
				}
			})
			if !a.moreOpen && !finished {
				// A press outside or Escape closed it: ease it out.
				a.moreOpen, a.moreTarget = true, 0
				c.Invalidate()
			}
		}
	})
}

// closeMore eases the More menu out, keeping it built until it fades.
func (a *app) closeMore(c *ui.Context) {
	if a.moreTarget == 0 {
		return // already closing
	}
	a.moreOpen = true
	a.moreTarget = 0
	c.Invalidate()
}

// menuItem draws a row of the More menu, which it reports a click of. A
// row that is disabled is muted and takes no click.
func menuItem(c *ui.Context, icon *ui.SVG, label string, enabled bool) bool {
	t := c.Theme()
	row := ui.Row(c).Gap(8).AlignItems(ui.Center).Padding(7, 10).Radius(9).
		Label(label)
	fg := t.Text
	if enabled {
		row.Cursor(ui.CursorPointer)
		if row.Hovered() {
			row.Background(t.Accent.Alpha(0.16))
		}
	} else {
		fg = t.TextMuted.Alpha(0.7)
	}
	row.Children(func() {
		ui.Icon(c, icon).FontSize(14).TextColor(fg)
		ui.Text(c, label).FontSize(13).TextColor(fg).SingleLine()
	})
	return enabled && row.Clicked()
}

// menuSeparator draws a hairline between groups of rows.
func menuSeparator(c *ui.Context) {
	ui.Box(c).Height(1).Margin(4, 8).Background(c.Theme().Border.Alpha(0.7))
}

func main() {
	// One window: starting the app again brings the running one forward.
	if !mygo.App.RequestSingleInstanceLock() {
		return
	}
	// Check the signed releases for a newer version and install it.
	mygo.Use(native.Plugin)
	a := &app{}
	mygo.App.OnSecondInstance(func([]string, string) {
		if a.win != nil {
			a.win.Restore()
			a.win.Focus()
		}
	})
	a.load()
	mygo.App.WhenReady(func() {
		win := mygo.NewWindow(mygo.WindowOptions{
			Title:         "winswx",
			TitleBarStyle: mygo.TitleBarHidden,
			Width:         640,
			Height:        480,
			MinWidth:      320,
			MinHeight:     280,
			// Opens where the user left it last time.
			StateKey: "main",
			// The window shows the interface MyGo draws, not a web page.
			Content: ui.View(a.view),
		})
		a.win = win
		// Services added or removed on disk, and their states, show when
		// the window comes back to the front.
		win.OnFocus(func() { win.Update(a.load) })
		a.refreshStatuses()
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
