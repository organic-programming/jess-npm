package service

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"
)

var expectedRPCs = []string{
	"Init",
	"Install",
	"Uninstall",
	"Update",
	"Add",
	"RunScript",
	"Test",
	"Build",
	"NodeExec",
	"ListDeps",
	"Audit",
	"Outdated",
	"ReadPackageJson",
	"Pack",
	"Publish",
	"CacheClean",
	"Npx",
	"Version",
}

func TestRPCSurfaceParity(t *testing.T) {
	root := repoRoot(t)
	holonDir := filepath.Join(root, "holons", "jess-npm")

	protoRPCs := extractRPCs(t, filepath.Join(root, "_protos", "npm", "v1", "npm.proto"))
	if !slices.Equal(protoRPCs, expectedRPCs) {
		t.Fatalf("proto RPCs = %v, want %v", protoRPCs, expectedRPCs)
	}

	manifestRPCs := extractManifestRPCs(t, filepath.Join(holonDir, "api", "v1", "holon.proto"))
	if !slices.Equal(manifestRPCs, expectedRPCs) {
		t.Fatalf("manifest contract.rpcs = %v, want %v", manifestRPCs, expectedRPCs)
	}

	serviceMethods := extractServiceMethods(t, filepath.Join(holonDir, "internal", "service", "service.go"))
	if !slices.Equal(serviceMethods, expectedRPCs) {
		t.Fatalf("service methods = %v, want %v", serviceMethods, expectedRPCs)
	}

	testMentions := extractRPCMentionsFromTests(t, holonDir)
	if !slices.Equal(testMentions, expectedRPCs) {
		t.Fatalf("test RPC mentions = %v, want %v", testMentions, expectedRPCs)
	}
}

func TestOPInspectMatchesRPCSurface(t *testing.T) {
	if _, err := exec.LookPath("op"); err != nil {
		t.Skip("op not found in PATH")
	}

	root := repoRoot(t)
	cmd := exec.Command("op", "inspect", "jess-npm", "--json")
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("op inspect jess-npm --json: %v", err)
	}

	var doc struct {
		Document struct {
			Services []struct {
				Name    string `json:"name"`
				Methods []struct {
					Name string `json:"name"`
				} `json:"methods"`
			} `json:"services"`
		} `json:"document"`
	}
	if err := json.Unmarshal(output, &doc); err != nil {
		t.Fatalf("unmarshal op inspect output: %v", err)
	}

	for _, service := range doc.Document.Services {
		if service.Name != "npm.v1.NpmService" {
			continue
		}

		methods := make([]string, 0, len(service.Methods))
		for _, method := range service.Methods {
			methods = append(methods, method.Name)
		}

		if !slices.Equal(methods, expectedRPCs) {
			t.Fatalf("op inspect methods = %v, want %v", methods, expectedRPCs)
		}
		return
	}

	t.Fatal("npm.v1.NpmService not found in op inspect output")
}

func repoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
}

func extractRPCs(t *testing.T, path string) []string {
	t.Helper()
	content := mustReadFile(t, path)
	matches := regexp.MustCompile(`(?m)^\s*rpc\s+(\w+)\s*\(`).FindAllStringSubmatch(content, -1)
	rpcs := make([]string, 0, len(matches))
	for _, match := range matches {
		rpcs = append(rpcs, match[1])
	}
	return rpcs
}

func extractManifestRPCs(t *testing.T, path string) []string {
	t.Helper()
	content := mustReadFile(t, path)
	match := regexp.MustCompile(`(?s)rpcs:\s*\[(.*?)\]`).FindStringSubmatch(content)
	if len(match) != 2 {
		t.Fatalf("rpcs block not found in %s", path)
	}

	names := regexp.MustCompile(`"(\w+)"`).FindAllStringSubmatch(match[1], -1)
	rpcs := make([]string, 0, len(names))
	for _, name := range names {
		rpcs = append(rpcs, name[1])
	}
	return rpcs
}

func extractServiceMethods(t *testing.T, path string) []string {
	t.Helper()
	content := mustReadFile(t, path)
	matches := regexp.MustCompile(`(?m)^func \(s \*NpmServer\) (\w+)\(`).FindAllStringSubmatch(content, -1)
	methods := make([]string, 0, len(matches))
	for _, match := range matches {
		methods = append(methods, match[1])
	}
	return methods
}

func extractRPCMentionsFromTests(t *testing.T, holonDir string) []string {
	t.Helper()

	var builder strings.Builder
	err := filepath.WalkDir(holonDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return nil
		}

		builder.WriteString(mustReadFile(t, path))
		builder.WriteByte('\n')
		return nil
	})
	if err != nil {
		t.Fatalf("walk test files: %v", err)
	}

	content := builder.String()
	mentioned := make([]string, 0, len(expectedRPCs))
	for _, rpc := range expectedRPCs {
		if strings.Contains(content, rpc) {
			mentioned = append(mentioned, rpc)
		}
	}
	return mentioned
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
