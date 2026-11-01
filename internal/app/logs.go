package app

import (
	"io"
	"math"
	"mygo/internal/locales"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"
)

// openLogsDialog shows the log files of a service.
func (a *App) openLogsDialog(name string) {
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
func (a *App) reloadLogs() {
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
func (a *App) logLabel(name string) string {
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
func (a *App) logsForm(c *ui.Context, w, winH float32) {
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
