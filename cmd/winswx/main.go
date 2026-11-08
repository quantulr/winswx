//go:generate go run ../../scripts/fetch-winsw

// Command winswx starts the service manager.
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/updater/native"

	"mygo/internal/app"
)

func main() {
	// One window: starting the app again brings the running one forward.
	if !mygo.App.RequestSingleInstanceLock() {
		return
	}
	// Check the signed releases for a newer version and install it. The
	// update window and its notes follow the app's language.
	mygo.Updater.SetSource(localizedFeed{base: updateFeed, locale: mygo.App.Locale})
	mygo.Use(native.Plugin)
	a := app.New()
	mygo.App.OnSecondInstance(func([]string, string) { a.Focus() })
	mygo.App.WhenReady(a.OpenWindow)
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
