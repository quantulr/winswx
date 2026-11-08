package app

import (
	"mygo/internal/locales"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestFirewallNoticeShowsOnFirstRun(t *testing.T) {
	a := &App{dataDir: t.TempDir(), firewallNotice: true}
	tt := ui.NewTester(a.view, 480, 360)
	for _, want := range []string{
		locales.FirewallNoticeTitle(),
		locales.FirewallNoticeMessage(),
		locales.FirewallNoticeAction(),
		locales.GotIt(),
	} {
		if !tt.HasText(want) {
			t.Errorf("texts %q do not show %q", tt.Texts(), want)
		}
	}
}

func TestFirewallNoticeGotItRemembers(t *testing.T) {
	dir := t.TempDir()
	a := &App{dataDir: dir, firewallNotice: true}
	tt := ui.NewTester(a.view, 480, 360)
	if err := tt.Click(locales.GotIt()); err != nil {
		t.Fatal(err)
	}
	if a.firewallNotice {
		t.Error("the notice is still open")
	}
	if _, err := os.Stat(filepath.Join(dir, firewallNoticeFile)); err != nil {
		t.Errorf("the notice was not remembered: %v", err)
	}
}

func TestFirewallNoticeOpensSettings(t *testing.T) {
	dir := t.TempDir()
	a := &App{dataDir: dir, firewallNotice: true}
	tt := ui.NewTester(a.view, 480, 360)
	if err := tt.Click(locales.FirewallNoticeAction()); err != nil {
		t.Fatal(err)
	}
	if got := tt.OpenedURLs(); len(got) != 1 || got[0] != firewallSettingsURL {
		t.Errorf("opened %q, want [%q]", got, firewallSettingsURL)
	}
	if a.firewallNotice {
		t.Error("the notice is still open")
	}
	if _, err := os.Stat(filepath.Join(dir, firewallNoticeFile)); err != nil {
		t.Errorf("the notice was not remembered: %v", err)
	}
}

func TestFirewallNoticeSeen(t *testing.T) {
	dir := t.TempDir()
	a := &App{dataDir: dir}
	if a.firewallNoticeSeen() {
		t.Error("a fresh data directory has seen the notice")
	}
	if err := os.WriteFile(filepath.Join(dir, firewallNoticeFile), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if !a.firewallNoticeSeen() {
		t.Error("the notice file does not count as seen")
	}
	if !(&App{}).firewallNoticeSeen() {
		t.Error("without a data directory the notice counts as seen")
	}
}
