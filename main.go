package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"
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
}

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
func (a *app) run(name, command string) {
	if a.win == nil || a.winsw == "" || a.busy[name] {
		return
	}
	config := serviceConfig(a.dir, name)
	if config == "" {
		a.opErr = name + ": no WinSW configuration file"
		return
	}
	if a.busy == nil {
		a.busy = make(map[string]bool)
	}
	a.busy[name] = true
	a.opErr = ""
	winsw, win := a.winsw, a.win
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
					ui.Text(c, "Services").FontSize(20).Bold()
					ui.Textf(c, "%d", len(a.services)).TextColor(t.TextMuted)
					ui.Spacer(c).Grow(1)
					refresh := ui.Button(c, "").Label("Refresh").Tooltip("Refresh").
						Padding(6, 8).BorderWidth(0).Radius(10).Cursor(ui.CursorPointer).
						Material(glass.Glass{Interactive: true}).Children(func() {
						ui.Icon(c, refreshIcon).FontSize(16)
					})
					if refresh.Clicked() {
						a.load()
					}
				})
				if a.opErr != "" {
					ui.Text(c, a.opErr).FontSize(12).TextColor(t.Danger).Wrap()
				}
				switch {
				case a.err != nil:
					ui.Text(c, a.err.Error()).TextColor(t.Danger)
				case len(a.services) == 0:
					ui.Column(c).Grow(1).Center().Children(func() {
						ui.Text(c, "No services yet.").TextColor(t.TextMuted)
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
							for _, s := range a.services {
								a.serviceCard(c, s)
							}
						})
					})
				}
				ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Spacer(c).Grow(1)
					if a.winsw == "" {
						ui.Text(c, "winsw.exe missing").TextColor(t.Danger).FontSize(12)
					}
				})
			})
		})
	})
}

// serviceCard draws a service as a card in the grid, with its first
// letter as a badge, its status under its name, and its actions below.
func (a *app) serviceCard(c *ui.Context, s service) {
	t := c.Theme()
	busy := a.busy[s.name]
	status := s.status
	color := statusColor(t, status)
	if busy {
		status, color = "Working…", t.Accent
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
			ui.Box(c).Width(34).Height(34).Radius(9).Center().
				Background(t.Accent.Alpha(0.12)).Children(func() {
				ui.Text(c, initial).FontSize(16).Bold().TextColor(t.Accent)
			})
			ui.Column(c).Gap(2).Children(func() {
				ui.Text(c, s.name).Bold().SingleLine()
				ui.Row(c).AlignItems(ui.Center).Gap(5).Children(func() {
					ui.Box(c).Width(7).Height(7).Radius(4).Background(color)
					ui.Text(c, string(status)).FontSize(12).TextColor(color).SingleLine()
				})
			})
		})
		a.serviceButtons(c, s, busy)
	})
}

// serviceButtons shows the actions of a service, by its status.
func (a *app) serviceButtons(c *ui.Context, s service, busy bool) {
	installed := s.status == statusRunning || s.status == statusStopped ||
		s.status == statusStarting || s.status == statusStopping || s.status == statusPaused
	ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
		label, command := "", ""
		switch s.status {
		case statusRunning, statusPaused, statusStarting:
			label, command = "Stop", "stop"
		case statusStopped:
			label, command = "Start", "start"
		case statusNotInstalled:
			label, command = "Install", "install"
		}
		if label != "" {
			var b ui.Element
			if command == "stop" {
				b = ui.Button(c, label)
			} else {
				b = ui.PrimaryButton(c, label)
			}
			if b.Padding(3, 7).Cursor(ui.CursorPointer).Disabled(busy).Clicked() {
				a.run(s.name, command)
			}
		}
		if installed {
			ui.Button(c, "More").Padding(3, 7).Cursor(ui.CursorPointer).
				Disabled(busy).Menu(func(m *ui.Menu) {
				if m.Item("Restart").Disabled(busy).Chosen() {
					a.run(s.name, "restart")
				}
				m.Separator()
				if m.Item("Uninstall").Disabled(busy || s.status != statusStopped).Chosen() {
					a.run(s.name, "uninstall")
				}
			})
		}
	})
}

func main() {
	a := &app{}
	a.load()
	mygo.App.WhenReady(func() {
		win := mygo.NewWindow(mygo.WindowOptions{
			Title:         "winswx",
			TitleBarStyle: mygo.TitleBarHidden,
			Width:         480,
			Height:        360,
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
