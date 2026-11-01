package app

import (
	"mygo/internal/locales"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestViewShowsServiceActions(t *testing.T) {
	a := &App{services: []service{{name: "caddy", status: statusStopped}}}
	tt := ui.NewTester(a.view, 480, 360)
	for _, label := range []string{"caddy", statusLabel(statusStopped), locales.Start(), locales.More()} {
		if !tt.HasText(label) {
			t.Errorf("texts %q do not show %q", tt.Texts(), label)
		}
	}

	a = &App{services: []service{{name: "new", status: statusNotInstalled}}}
	tt = ui.NewTester(a.view, 480, 360)
	for _, label := range []string{"new", statusLabel(statusNotInstalled), locales.Install(), locales.More()} {
		if !tt.HasText(label) {
			t.Errorf("texts %q do not show %q", tt.Texts(), label)
		}
	}
}

func TestCardWidthFixed(t *testing.T) {
	a := &App{services: []service{
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
	a := &App{services: []service{{name: "caddy", status: statusStopped}}}
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

func TestAddDialogOpens(t *testing.T) {
	a := &App{}
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
	a := &App{dir: t.TempDir(), winsw: "winsw.exe"}
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

func TestDeleteServiceFromMenu(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(dir, "demo", "demo.xml")
	if err := os.WriteFile(config, []byte("<service><id>demo</id><executable>x.exe</executable></service>"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &App{
		dir:      dir,
		winsw:    "winsw.exe",
		services: []service{{name: "demo", status: statusNotInstalled}},
	}
	tt := ui.NewTester(a.view, 480, 360)
	if err := tt.Click(locales.More()); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(locales.Delete()); err != nil {
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

func TestMoreMenuAnimates(t *testing.T) {
	a := &App{dir: t.TempDir(), winsw: "winsw.exe",
		services: []service{{name: "caddy", status: statusStopped}}}
	tt := ui.NewTester(a.view, 480, 360)
	if err := tt.Click(locales.More()); err != nil {
		t.Fatal(err)
	}
	tt.Advance(250 * time.Millisecond)
	if !a.moreOpen {
		t.Fatal("the menu did not open")
	}
	if err := tt.Click(locales.More()); err != nil {
		t.Fatal(err)
	}
	tt.Advance(40 * time.Millisecond)
	if !a.moreOpen {
		t.Fatal("the menu was removed before its closing animation")
	}
	tt.Advance(300 * time.Millisecond)
	if a.moreOpen {
		t.Fatal("the menu stayed after its closing animation")
	}
}

func TestMoreMenuClosesOutside(t *testing.T) {
	a := &App{dir: t.TempDir(), winsw: "winsw.exe",
		services: []service{{name: "caddy", status: statusStopped}}}
	tt := ui.NewTester(a.view, 480, 360)
	if err := tt.Click(locales.More()); err != nil {
		t.Fatal(err)
	}
	tt.Advance(250 * time.Millisecond)
	if !a.moreOpen {
		t.Fatal("the menu did not open")
	}
	tt.ClickAt(30, 60)
	tt.Advance(300 * time.Millisecond)
	if a.moreOpen {
		t.Fatal("a press outside did not close the menu")
	}
}

func TestMoreMenuClosesOnEscape(t *testing.T) {
	a := &App{dir: t.TempDir(), winsw: "winsw.exe",
		services: []service{{name: "caddy", status: statusStopped}}}
	tt := ui.NewTester(a.view, 480, 360)
	if err := tt.Click(locales.More()); err != nil {
		t.Fatal(err)
	}
	tt.Advance(250 * time.Millisecond)
	tt.Key(0, ui.KeyEscape)
	tt.Advance(300 * time.Millisecond)
	if a.moreOpen {
		t.Fatal("Escape did not close the menu")
	}
}
