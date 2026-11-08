package app

import (
	"errors"
	"io"
	"mygo/internal/locales"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// app is the state the window shows. Its view builds the interface from
// it, on the main thread, whenever the window needs a frame: after input,
// after Window.Update, and while something animates.
type App struct {
	dir      string          // the services directory, in the user data directory
	dataDir  string          // the user data directory, "" when unavailable
	services []service       // the directories in dir: a service each
	winsw    string          // the bundled WinSW executable, "" when it is missing
	busy     map[string]bool // the services with a WinSW command running
	opErr    string          // why the last operation failed
	err      error           // why the services could not be read, if they could not
	win      *mygo.Window    // the window, nil before it is made
	queryGen int             // counts status queries, so a stale one is ignored

	dialog         dialogMode // the dialog that shows, none when closed
	editDir        string     // the directory of the service being edited
	moreName       string     // the service whose More menu is open
	moreOpen       bool       // the More menu is built
	moreTarget     float32    // 1 shown, 0 closing
	moreFresh      bool       // seed the opening animation
	deleting       string     // the service asked about for deletion
	confirmDelete  bool       // the delete alert shows
	firewallNotice bool       // the first-run firewall notice shows
	logsDir        string     // the service whose logs the dialog shows
	logsOptions    []string   // the log files the segmented control offers
	logsExisting   []bool     // whether each option's file exists
	logsFile       string     // the log file shown
	logsText       string     // its tail
	logsTrunc      bool       // the tail is not the whole file
	logsErr        string     // why the logs could not be read
	logsTab        int        // the log chosen in the segmented control
	logsScroll     ui.ScrollState
	dlgOpen        bool    // DialogBase's open flag
	dlgTarget      float32 // 1 shown, 0 closing
	dlgFresh       bool    // seed the opening animation
	newName        string  // the name in the dialog
	newExe         string  // the executable in the dialog
	newArgs        string  // the arguments in the dialog
	dlgErr         string  // why the dialog could not be saved
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

// load creates the services directory in the user data directory if it is
// missing, then lists the directories in it. Each directory is a service.
// The statuses are queried in the background.
func (a *App) load() {
	a.dataDir = ""
	userData, err := mygo.App.Path(mygo.PathUserData)
	if err == nil {
		a.dataDir = userData
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
func (a *App) refreshStatuses() {
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
func (a *App) serviceWinsw() (string, error) {
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

func (a *App) run(name, command string) {
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

// New returns the app, with its services loaded.
func New() *App {
	a := &App{}
	a.load()
	if runtime.GOOS == "windows" && a.err == nil {
		a.firewallNotice = !a.firewallNoticeSeen()
	}
	return a
}

// Focus brings the window forward, as a second launch asks.
func (a *App) Focus() {
	if a.win == nil {
		return
	}
	a.win.Restore()
	a.win.Focus()
}

// OpenWindow shows the app's window once MyGo is ready.
func (a *App) OpenWindow() {
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
	// Services added or removed on disk, and their states, show when the
	// window comes back to the front.
	win.OnFocus(func() { win.Update(a.load) })
	a.refreshStatuses()
}
