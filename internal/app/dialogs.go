package app

import (
	"encoding/xml"
	"mygo/internal/locales"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// dialogMode names the form the dialog shows.
type dialogMode uint8

// The forms of the dialog.
const (
	dialogNone dialogMode = iota
	dialogAdd
	dialogEdit
	dialogLogs
)

// addService creates a service directory and its WinSW configuration from
// the fields of the Add Service dialog.
func (a *App) addService() bool {
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

// dialogView shows the Add or Edit Service form, easing it in and out.
func (a *App) dialogView(c *ui.Context) {
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
func (a *App) openAddDialog() {
	a.dialog = dialogAdd
	a.dlgOpen = true
	a.dlgTarget = 0
	a.dlgFresh = true
	a.dlgErr = ""
	a.newName, a.newExe, a.newArgs = "", "", ""
}

// openEditDialog shows the form of an existing service, filled in from
// its WinSW configuration.
func (a *App) openEditDialog(name string) {
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
func (a *App) closeDialog(c *ui.Context) {
	if a.dlgTarget == 0 {
		return
	}
	a.dlgOpen = true
	a.dlgTarget = 0
	c.Invalidate()
}

// deleteAlert asks before removing a service and its files.
func (a *App) deleteAlert(c *ui.Context) {
	switch ui.AlertDialog(c, &a.confirmDelete, locales.DeleteTitle(a.deleting),
		locales.DeleteMessage(), locales.Cancel(), locales.DeleteAction()) {
	case 1:
		a.deleteService()
	}
}

// deleteService removes the directory of the service being deleted.
func (a *App) deleteService() {
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
func (a *App) saveService() bool {
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

// pickExecutable asks for the executable of the service with the system's
// file dialog.
func (a *App) pickExecutable() {
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
