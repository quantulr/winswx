package app

import (
	"mygo/internal/locales"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"
)

// Cards keep their width as the window resizes: the grid adds or drops
// columns instead.
const (
	cardWidth  = 200
	cardHeight = 112
)

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

func (a *App) view(c *ui.Context) {
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
	a.firewallNoticeAlert(c)
}

// serviceCard draws a service as a card in the grid, with its first
// letter as a badge, its status under its name, and its actions below.
func (a *App) serviceCard(c *ui.Context, i int, s service) {
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

func (a *App) serviceButtons(c *ui.Context, s service, busy bool) {
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
func (a *App) closeMore(c *ui.Context) {
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
