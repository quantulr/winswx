package app

import (
	"os"
	"testing"

	"github.com/egoist/mygo"
)

// TestMain points PathResources at the project's resources: go test runs
// in the package directory, while mygo build and mygo dev put them at the
// project root.
func TestMain(m *testing.M) {
	mygo.App.SetPath(mygo.PathResources, "../../resources")
	os.Exit(m.Run())
}
