package app

import (
	"fmt"
	"mygo/internal/locales"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestReadLogTail(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.log")
	if err := os.WriteFile(path, []byte("aa\nbb\ncc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, trunc, err := readLogTail(path, 100)
	if err != nil || trunc || got != "aa\nbb\ncc\n" {
		t.Fatalf("whole file = %q, trunc %v, %v", got, trunc, err)
	}
	got, trunc, err = readLogTail(path, 5)
	if err != nil || !trunc || got != "cc\n" {
		t.Fatalf("tail = %q, trunc %v, %v", got, trunc, err)
	}
}

func TestOpenLogs(t *testing.T) {
	dir := t.TempDir()
	serviceDir := filepath.Join(dir, "demo")
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(serviceDir, "demo.out.log"),
		[]byte("old\nlast line\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(serviceDir, "demo.err.log"),
		[]byte("error line\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &App{dir: dir}
	a.openLogsDialog("demo")
	if a.dialog != dialogLogs {
		t.Fatal("the logs dialog did not open")
	}
	if a.logsFile != "demo.out.log" {
		t.Fatalf("selected %q, want the output log", a.logsFile)
	}
	if !strings.Contains(a.logsText, "last line") {
		t.Errorf("the log tail %q does not hold the last line", a.logsText)
	}
	tt := ui.NewTester(a.view, 480, 360)
	if !tt.HasText("last line") {
		t.Errorf("texts %q do not show the log", tt.Texts())
	}

	// Choosing the other segment reads the other log.
	a.logsTab = 2 // demo.err.log, the third option
	tt.Frame()
	if a.logsFile != "demo.err.log" {
		t.Fatalf("the segmented control chose %q", a.logsFile)
	}
	if !strings.Contains(a.logsText, "error line") {
		t.Errorf("the error log tail = %q", a.logsText)
	}
}

func TestLogsKeepHeight(t *testing.T) {
	dir := t.TempDir()
	serviceDir := filepath.Join(dir, "demo")
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	var big strings.Builder
	for i := 0; i < 4000; i++ {
		fmt.Fprintf(&big, "line %d with enough words to fill it\n", i)
	}
	if err := os.WriteFile(filepath.Join(serviceDir, "demo.out.log"),
		[]byte(big.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(serviceDir, "demo.err.log"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	a := &App{dir: dir}
	a.openLogsDialog("demo")
	tt := ui.NewTester(a.view, 480, 360)
	tt.Advance(300 * time.Millisecond)
	before, ok := tt.Find(locales.Close())
	if !ok {
		t.Fatal("Close not found")
	}
	a.logsTab = 2 // demo.err.log, empty
	tt.Frame()
	after, ok := tt.Find(locales.Close())
	if !ok {
		t.Fatal("Close not found after switching logs")
	}
	if before != after {
		t.Errorf("the dialog moved: Close at %v, then %v", before, after)
	}
}

func TestLogsSingleFile(t *testing.T) {
	dir := t.TempDir()
	serviceDir := filepath.Join(dir, "demo")
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(serviceDir, "demo.err.log"),
		[]byte("boom\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &App{dir: dir}
	a.openLogsDialog("demo")
	want := []string{"demo.wrapper.log", "demo.out.log", "demo.err.log"}
	if len(a.logsOptions) != len(want) {
		t.Fatalf("options = %v, want %v", a.logsOptions, want)
	}
	for i, name := range want {
		if a.logsOptions[i] != name {
			t.Errorf("option %d = %q, want %q", i, a.logsOptions[i], name)
		}
	}
	if a.logsExisting[0] || a.logsExisting[1] || !a.logsExisting[2] {
		t.Errorf("existing = %v, want only the error log", a.logsExisting)
	}
	if a.logsFile != "demo.err.log" || a.logsTab != 2 {
		t.Errorf("chose %q at %d, want the error log", a.logsFile, a.logsTab)
	}

	tt := ui.NewTester(a.view, 480, 360)
	for _, label := range []string{"out.log", "err.log", "wrapper.log"} {
		if !tt.HasText(label) {
			t.Errorf("texts %q do not name %q", tt.Texts(), label)
		}
	}
	if !tt.HasText("boom") {
		t.Errorf("texts %q do not show the log", tt.Texts())
	}
}

func TestLogsNoFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	a := &App{dir: dir}
	a.openLogsDialog("demo")
	if a.logsFile != "" {
		t.Fatalf("chose %q, want no log", a.logsFile)
	}
	tt := ui.NewTester(a.view, 480, 360)
	if !tt.HasText(locales.NoLogs()) {
		t.Errorf("texts %q do not show the empty state", tt.Texts())
	}
	for _, label := range []string{"out.log", "err.log", "wrapper.log"} {
		if tt.HasText(label) {
			t.Errorf("texts %q show %q without any log file", tt.Texts(), label)
		}
	}
}

func TestLogsWrapperDefault(t *testing.T) {
	dir := t.TempDir()
	serviceDir := filepath.Join(dir, "demo")
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"demo.out.log", "demo.err.log", "demo.wrapper.log"} {
		if err := os.WriteFile(filepath.Join(serviceDir, name), []byte(name+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	a := &App{dir: dir}
	a.openLogsDialog("demo")
	if a.logsFile != "demo.wrapper.log" || a.logsTab != 0 {
		t.Fatalf("chose %q at %d, want the wrapper log", a.logsFile, a.logsTab)
	}
	if !strings.Contains(a.logsText, "demo.wrapper.log") {
		t.Errorf("the tail = %q", a.logsText)
	}
}
