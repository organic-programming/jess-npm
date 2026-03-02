package npmrunner

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestRunNpmVersion(t *testing.T) {
	requireBinaries(t, "npm")

	result := RunNpm([]string{"--version"}, ".", nil, 10)
	if result.ExitCode != 0 {
		t.Fatalf("exit=%d stderr=%q", result.ExitCode, result.Stderr)
	}

	version := strings.TrimSpace(result.Stdout)
	if !regexp.MustCompile(`^\d+\.\d+\.\d+`).MatchString(version) {
		t.Fatalf("stdout=%q, want semantic version", result.Stdout)
	}
}

func TestRunNodeVersion(t *testing.T) {
	requireBinaries(t, "node")

	result := RunNode([]string{"--version"}, ".", nil, 10)
	if result.ExitCode != 0 {
		t.Fatalf("exit=%d stderr=%q", result.ExitCode, result.Stderr)
	}
	if !strings.HasPrefix(strings.TrimSpace(result.Stdout), "v") {
		t.Fatalf("stdout=%q, want prefix v", result.Stdout)
	}
}

func TestRunNodeEval(t *testing.T) {
	requireBinaries(t, "node")

	result := RunNodeEval("console.log(1+1)", ".", nil, 10)
	if result.ExitCode != 0 {
		t.Fatalf("exit=%d stderr=%q", result.ExitCode, result.Stderr)
	}
	if strings.TrimSpace(result.Stdout) != "2" {
		t.Fatalf("stdout=%q, want 2", result.Stdout)
	}
}

func TestRunNpmBadDir(t *testing.T) {
	requireBinaries(t, "npm")

	result := RunNpm([]string{"--version"}, filepath.Join(t.TempDir(), "missing"), nil, 5)
	if result.ExitCode == 0 {
		t.Fatalf("expected non-zero exit, got 0 stdout=%q stderr=%q", result.Stdout, result.Stderr)
	}
}

func TestRunNpmTimeout(t *testing.T) {
	requireBinaries(t, "npm", "node")

	dir := t.TempDir()
	pkgJSON := `{
  "name": "timeout-test",
  "version": "1.0.0",
  "private": true,
  "scripts": {
    "slow": "node -e \"setTimeout(() => {}, 5000)\""
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkgJSON), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	result := RunNpm([]string{"run", "slow"}, dir, nil, 1)
	if result.ExitCode == 0 {
		t.Fatalf("expected timeout failure, got stdout=%q", result.Stdout)
	}
	if result.ExitCode != 124 {
		t.Fatalf("exit=%d, want 124", result.ExitCode)
	}
}

func requireBinaries(t *testing.T, binaries ...string) {
	t.Helper()
	for _, bin := range binaries {
		if _, err := exec.LookPath(bin); err != nil {
			t.Skipf("%s not found in PATH", bin)
		}
	}
}
