// Command fetch-winsw downloads the WinSW executable the app ships in its
// resources. The repository ignores it, so fetch it before the app is
// built or its tests run:
//
//	go run ./scripts/fetch-winsw
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// The WinSW release the app ships, and the SHA-256 of its x64 executable.
const (
	url     = "https://github.com/winsw/winsw/releases/download/v3.0.0-alpha.11/WinSW-x64.exe"
	sum     = "a2daa6a33a9c2b791ae31d9092e7935c339d1e03e89bfb747618ce2f4e819e20"
	dest    = "resources/bin/winsw.exe"
	version = "WinSW v3.0.0-alpha.11"
)

func main() {
	if err := fetch(); err != nil {
		fmt.Fprintln(os.Stderr, "fetch-winsw:", err)
		os.Exit(1)
	}
}

func fetch() error {
	if data, err := os.ReadFile(dest); err == nil {
		if sum256(data) == sum {
			fmt.Println(dest, "is up to date")
			return nil
		}
		return fmt.Errorf("%s exists but is not %s; remove it and run this again", dest, version)
	}
	fmt.Println("Downloading", url)
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s", url, resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if got := sum256(data); got != sum {
		return fmt.Errorf("downloaded %s has SHA-256 %s, want %s", url, got, sum)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	tmp := dest + ".download"
	if err := os.WriteFile(tmp, data, 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	fmt.Println("Saved", dest)
	return nil
}

func sum256(data []byte) string {
	s := sha256.Sum256(data)
	return hex.EncodeToString(s[:])
}
