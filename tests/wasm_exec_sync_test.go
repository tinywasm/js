package js_test

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/tinywasm/tinygo"
)

func TestWasmExecAnnotationPresent(t *testing.T) {
	check := func(filename, prefix string) {
		data, err := os.ReadFile("../assets/" + filename)
		if err != nil {
			t.Fatalf("failed to read %s: %v", filename, err)
		}
		if !strings.HasPrefix(string(data), prefix) {
			t.Errorf("%s missing expected annotation prefix %q", filename, prefix)
		}
	}

	check("wasm_exec_tinygo.js", "// @tinygo-version ")
	check("wasm_exec_go.js", "// @go-version ")
}

func TestWasmExecTinyGoInSync(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	want := tinygo.DefaultVersion
	data, err := os.ReadFile("../assets/wasm_exec_tinygo.js")
	if err != nil {
		t.Fatalf("failed to read asset: %v", err)
	}

	lines := strings.SplitN(string(data), "\n", 2)
	have := ""
	if strings.HasPrefix(lines[0], "// @tinygo-version ") {
		have = strings.TrimSpace(lines[0][len("// @tinygo-version "):])
	}

	url := fmt.Sprintf("https://github.com/tinygo-org/tinygo/releases/download/v%s/tinygo%s.linux-amd64.tar.gz", want, want)
	remoteData, err := downloadAndExtract(url, "tinygo/targets/wasm_exec.js")
	if err != nil {
		t.Fatalf("failed to download remote version: %v", err)
	}

	contentOnly := ""
	if len(lines) > 1 {
		contentOnly = lines[1]
	}

	if have != want || hash(contentOnly) != hash(string(remoteData)) {
		newContent := fmt.Sprintf("// @tinygo-version %s\n%s", want, string(remoteData))
		if err := os.WriteFile("../assets/wasm_exec_tinygo.js", []byte(newContent), 0644); err != nil {
			t.Fatalf("failed to update asset: %v", err)
		}
		t.Fatalf("wasm_exec_tinygo.js updated %s -> %s. Re-run: gotest ./...", have, want)
	}
}

func TestWasmExecGoInSync(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	want, err := getGoVersion()
	if err != nil {
		t.Fatalf("failed to get go version: %v", err)
	}

	data, err := os.ReadFile("../assets/wasm_exec_go.js")
	if err != nil {
		t.Fatalf("failed to read asset: %v", err)
	}

	lines := strings.SplitN(string(data), "\n", 2)
	have := ""
	if strings.HasPrefix(lines[0], "// @go-version ") {
		have = strings.TrimSpace(lines[0][len("// @go-version "):])
	}

	url := fmt.Sprintf("https://go.dev/dl/go%s.src.tar.gz", want)
	remoteData, err := downloadAndExtract(url, "go/lib/wasm/wasm_exec.js")
	if err != nil {
		t.Fatalf("failed to download remote version: %v", err)
	}

	contentOnly := ""
	if len(lines) > 1 {
		contentOnly = lines[1]
	}

	if have != want || hash(contentOnly) != hash(string(remoteData)) {
		newContent := fmt.Sprintf("// @go-version %s\n%s", want, string(remoteData))
		if err := os.WriteFile("../assets/wasm_exec_go.js", []byte(newContent), 0644); err != nil {
			t.Fatalf("failed to update asset: %v", err)
		}
		t.Fatalf("wasm_exec_go.js updated %s -> %s. Re-run: gotest ./...", have, want)
	}
}

func getGoVersion() (string, error) {
	data, err := os.ReadFile("../go.mod")
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

func hash(s string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(s)))
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
