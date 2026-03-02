package service

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/organic-programming/go-holons/pkg/grpcclient"
	"github.com/organic-programming/go-holons/pkg/transport"
	npmv1 "github.com/organic-programming/jess-npm/gen/go/npm/v1"
	"google.golang.org/grpc"
)

func startTestServer(t *testing.T) (npmv1.NpmServiceClient, func()) {
	t.Helper()

	mem := transport.NewMemListener()
	srv := grpc.NewServer()
	npmv1.RegisterNpmServiceServer(srv, &NpmServer{})
	go func() { _ = srv.Serve(mem) }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpcclient.DialMem(ctx, mem)
	if err != nil {
		t.Fatalf("DialMem: %v", err)
	}

	cleanup := func() {
		_ = conn.Close()
		srv.Stop()
		_ = mem.Close()
	}

	return npmv1.NewNpmServiceClient(conn), cleanup
}

func TestVersion(t *testing.T) {
	requireBinaries(t, "npm", "node")

	client, cleanup := startTestServer(t)
	defer cleanup()

	resp, err := client.Version(context.Background(), &npmv1.VersionRequest{})
	if err != nil {
		t.Fatalf("Version error: %v", err)
	}
	if resp.GetNpmVersion() == "" || resp.GetNodeVersion() == "" {
		t.Fatalf("invalid version response: %+v", resp)
	}
}

func TestReadPackageJson(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	dir := t.TempDir()
	content := `{
  "name": "service-test",
  "version": "0.1.0",
  "scripts": {"build": "node build.js"},
  "dependencies": {"left-pad": "1.3.0"}
}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	resp, err := client.ReadPackageJson(context.Background(), &npmv1.ReadPackageJsonRequest{Workdir: dir})
	if err != nil {
		t.Fatalf("ReadPackageJson error: %v", err)
	}
	if resp.GetName() != "service-test" || resp.GetVersion() != "0.1.0" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.GetScripts()["build"] == "" {
		t.Fatalf("missing build script: %+v", resp.GetScripts())
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
