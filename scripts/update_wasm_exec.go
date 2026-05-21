//go:build ignore

package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/tinywasm/tinygo"
)

func main() {
	if err := updateTinyGo(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to update tinygo: %v\n", err)
		os.Exit(1)
	}
	if err := updateGo(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to update go: %v\n", err)
		os.Exit(1)
	}
}

func updateTinyGo() error {
	version := tinygo.DefaultVersion
	url := fmt.Sprintf("https://github.com/tinygo-org/tinygo/releases/download/v%s/tinygo%s.linux-amd64.tar.gz", version, version)
	fmt.Printf("Updating TinyGo wasm_exec.js to %s...\n", version)

	data, err := downloadAndExtract(url, "tinygo/targets/wasm_exec.js")
	if err != nil {
		return err
	}

	content := fmt.Sprintf("// @tinygo-version %s\n%s", version, string(data))
	return os.WriteFile("assets/wasm_exec_tinygo.js", []byte(content), 0644)
}

func updateGo() error {
	version, err := getGoVersion()
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://go.dev/dl/go%s.src.tar.gz", version)
	fmt.Printf("Updating Go wasm_exec.js to %s...\n", version)

	data, err := downloadAndExtract(url, "go/lib/wasm/wasm_exec.js")
	if err != nil {
		return err
	}

	content := fmt.Sprintf("// @go-version %s\n%s", version, string(data))
	return os.WriteFile("assets/wasm_exec_go.js", []byte(content), 0644)
}

func getGoVersion() (string, error) {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "go ") {
			return strings.TrimSpace(line[3:]), nil
		}
	}
	return "", fmt.Errorf("go version not found in go.mod")
}

func downloadAndExtract(url, targetPath string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if header.Name == targetPath {
			return io.ReadAll(tr)
		}
	}

	return nil, fmt.Errorf("file %s not found in archive", targetPath)
}
