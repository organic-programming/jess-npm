package pkgjson

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadValid(t *testing.T) {
	dir := t.TempDir()
	content := `{
  "name": "demo",
  "version": "1.2.3",
  "description": "demo package",
  "main": "index.js",
  "license": "MIT",
  "scripts": {"test": "node test.js"},
  "dependencies": {"left-pad": "1.3.0"},
  "devDependencies": {"typescript": "5.9.0"},
  "peerDependencies": {"react": "^19.0.0"}
}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	pkg, err := Read(dir)
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	if pkg.Name != "demo" || pkg.Version != "1.2.3" {
		t.Fatalf("unexpected basic fields: %+v", pkg)
	}
	if pkg.Scripts["test"] == "" {
		t.Fatalf("missing script: %+v", pkg.Scripts)
	}
	if pkg.Dependencies["left-pad"] != "1.3.0" {
		t.Fatalf("unexpected dependency map: %+v", pkg.Dependencies)
	}
	if pkg.Raw == "" {
		t.Fatal("expected raw json")
	}
}

func TestReadMissing(t *testing.T) {
	_, err := Read(filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatal("expected error for missing package.json")
	}
}

func TestReadMalformed(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{invalid"), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	_, err := Read(dir)
	if err == nil {
		t.Fatal("expected json parse error")
	}
}
