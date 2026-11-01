package app

import (
	"bytes"
	"encoding/xml"
	"io"
	"mygo/internal/locales"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// serviceStatus is how Windows reports the state of a service.
type serviceStatus string

// The states a service shows.
const (
	statusChecking     serviceStatus = "Checking…"
	statusRunning      serviceStatus = "Running"
	statusStopped      serviceStatus = "Stopped"
	statusStarting     serviceStatus = "Starting"
	statusStopping     serviceStatus = "Stopping"
	statusPaused       serviceStatus = "Paused"
	statusNotInstalled serviceStatus = "Not installed"
	statusUnknown      serviceStatus = "Unknown"
)

// service is a directory in the services directory, and the status of the
// Windows service named after it.
type service struct {
	name   string
	status serviceStatus
}

// parseStatus maps what WinSW prints for a service to a serviceStatus:
// v3 prints systemd-style states, such as "Active (running)".
func parseStatus(out string) serviceStatus {
	switch strings.TrimSpace(out) {
	case "Active (running)":
		return statusRunning
	case "Active (starting)":
		return statusStarting
	case "Active (stopping)":
		return statusStopping
	case "Active (pausing)", "Active (paused)", "Active (continuing)":
		return statusPaused
	case "Inactive (stopped)":
		return statusStopped
	case "NonExistent":
		return statusNotInstalled
	}
	return statusUnknown
}

// queryService asks WinSW for the state of the service that config
// describes. WinSW prints the state to stdout even when it exits non-zero:
// v3.0.0 exits 1 for a running service, so the output is what counts.
func queryService(winsw, config string) serviceStatus {
	if winsw == "" || config == "" {
		return statusUnknown
	}
	cmd := exec.Command(winsw, "status", config)
	hideWindow(cmd)
	out, err := cmd.Output()
	if status := parseStatus(string(out)); status != statusUnknown {
		return status
	}
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok && exit.ExitCode() == 1060 {
			return statusNotInstalled
		}
	}
	return statusUnknown
}

// serviceConfig returns the WinSW configuration of the service in the
// directory dir/name: the XML named after it, or the only XML in it.
func serviceConfig(dir, name string) string {
	serviceDir := filepath.Join(dir, name)
	path := filepath.Join(serviceDir, name+".xml")
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		return path
	}
	entries, err := os.ReadDir(serviceDir)
	if err != nil {
		return ""
	}
	found := ""
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".xml") {
			continue
		}
		if found != "" {
			// More than one XML: not clear which one describes the service.
			return ""
		}
		found = filepath.Join(serviceDir, entry.Name())
	}
	return found
}

// serviceXML returns the WinSW configuration of a service.
func serviceXML(name, executable, arguments string) ([]byte, error) {
	cfg := struct {
		XMLName    xml.Name `xml:"service"`
		ID         string   `xml:"id"`
		Executable string   `xml:"executable"`
		Arguments  string   `xml:"arguments,omitempty"`
	}{ID: name, Executable: executable, Arguments: arguments}
	body, err := xml.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, err
	}
	out := append([]byte(xml.Header), body...)
	return append(out, '\n'), nil
}

// validateServiceName reports why name cannot be the name of a service
// directory, or "" when it can.
func validateServiceName(name string) string {
	switch {
	case name == "":
		return locales.ErrNeedsName()
	case name == "." || name == "..":
		return locales.ErrNameDots()
	case strings.TrimSpace(name) != name:
		return locales.ErrNameEndsInSpace()
	case strings.ContainsAny(name, `\/:*?"<>|`):
		return locales.ErrNameChars()
	}
	for _, r := range name {
		if r < 32 {
			return locales.ErrNameControl()
		}
	}
	base := name
	if i := strings.IndexByte(base, '.'); i >= 0 {
		base = base[:i]
	}
	switch strings.ToUpper(base) {
	case "CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return locales.ErrNameReserved()
	}
	return ""
}

// updateServiceXML returns WinSW configuration data with the id,
// executable and arguments replaced, keeping the other elements as they
// are.
func updateServiceXML(data []byte, id, executable, arguments string) ([]byte, error) {
	values := map[string]string{"id": id, "executable": executable, "arguments": arguments}
	dec := xml.NewDecoder(bytes.NewReader(data))
	var out bytes.Buffer
	out.WriteString(xml.Header)
	enc := xml.NewEncoder(&out)
	found := make(map[string]bool, len(values))
	depth := 0
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.ProcInst:
			if strings.EqualFold(t.Target, "xml") {
				continue // the header is already written
			}
		case xml.StartElement:
			if value, ok := values[t.Name.Local]; ok && depth == 1 {
				found[t.Name.Local] = true
				if t.Name.Local == "arguments" && arguments == "" {
					if err := dec.Skip(); err != nil {
						return nil, err
					}
					continue
				}
				if err := enc.EncodeToken(t); err != nil {
					return nil, err
				}
				if err := enc.EncodeToken(xml.CharData(value)); err != nil {
					return nil, err
				}
				if err := enc.EncodeToken(t.End()); err != nil {
					return nil, err
				}
				if err := dec.Skip(); err != nil {
					return nil, err
				}
				continue
			}
			depth++
		case xml.EndElement:
			depth--
			if depth == 0 {
				for _, name := range []string{"id", "executable", "arguments"} {
					if found[name] || (name == "arguments" && arguments == "") {
						continue
					}
					start := xml.StartElement{Name: xml.Name{Local: name}}
					_ = enc.EncodeToken(xml.CharData("\n  "))
					_ = enc.EncodeToken(start)
					_ = enc.EncodeToken(xml.CharData(values[name]))
					_ = enc.EncodeToken(start.End())
				}
			}
		}
		if err := enc.EncodeToken(tok); err != nil {
			return nil, err
		}
	}
	if err := enc.Flush(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
