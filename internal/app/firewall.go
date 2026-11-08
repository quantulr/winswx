package app

import (
	"mygo/internal/locales"
	"os"
	"path/filepath"

	"github.com/egoist/mygo/ui"
)

// firewallSettingsURL opens Windows Security's Firewall & network
// protection page, where a service that accepts connections is let
// through.
const firewallSettingsURL = "windowsdefender://network"

// firewallNoticeFile marks that the first-run firewall notice was shown.
const firewallNoticeFile = "firewall-notice"

// firewallNoticeSeen reports whether the notice was shown before.
func (a *App) firewallNoticeSeen() bool {
	if a.dataDir == "" {
		return true // no data directory to remember it in
	}
	_, err := os.Stat(filepath.Join(a.dataDir, firewallNoticeFile))
	return err == nil
}

// rememberFirewallNotice keeps the notice from showing again.
func (a *App) rememberFirewallNotice() {
	if a.dataDir == "" {
		return
	}
	_ = os.WriteFile(filepath.Join(a.dataDir, firewallNoticeFile), nil, 0o644)
}

// firewallNoticeAlert tells the user on first launch that a service that
// accepts connections needs a rule in Windows Firewall.
func (a *App) firewallNoticeAlert(c *ui.Context) {
	if !a.firewallNotice {
		return
	}
	switch ui.AlertDialog(c, &a.firewallNotice, locales.FirewallNoticeTitle(),
		locales.FirewallNoticeMessage(), locales.FirewallNoticeAction(), locales.GotIt()) {
	case -1:
		return // still open
	case 0:
		c.OpenURL(firewallSettingsURL)
	}
	a.rememberFirewallNotice()
}
