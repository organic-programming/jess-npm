package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	npmv1 "github.com/organic-programming/jess-npm/gen/go/npm/v1"
	"github.com/organic-programming/jess-npm/internal/npmrunner"
	"github.com/organic-programming/jess-npm/internal/pkgjson"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type NpmServer struct {
	npmv1.UnimplementedNpmServiceServer
}

func (s *NpmServer) Init(_ context.Context, req *npmv1.NpmCommandRequest) (*npmv1.NpmCommandResponse, error) {
	return runGenericNpm(req, "init"), nil
}

func (s *NpmServer) Install(_ context.Context, req *npmv1.InstallRequest) (*npmv1.NpmCommandResponse, error) {
	args := []string{"install"}
	args = append(args, req.GetPackages()...)
	if req.GetSaveDev() {
		args = append(args, "--save-dev")
	}
	if req.GetSaveExact() {
		args = append(args, "--save-exact")
	}
	if req.GetProduction() {
		args = append(args, "--omit=dev")
	}
	if req.GetLegacyPeerDeps() {
		args = append(args, "--legacy-peer-deps")
	}

	result := npmrunner.RunNpm(args, defaultWorkdir(req.GetWorkdir()), req.GetEnv(), int(req.GetTimeoutS()))
	return mapCommandResponse(result), nil
}

func (s *NpmServer) Uninstall(_ context.Context, req *npmv1.NpmCommandRequest) (*npmv1.NpmCommandResponse, error) {
	return runGenericNpm(req, "uninstall"), nil
}

func (s *NpmServer) Update(_ context.Context, req *npmv1.NpmCommandRequest) (*npmv1.NpmCommandResponse, error) {
	return runGenericNpm(req, "update"), nil
}

func (s *NpmServer) Add(_ context.Context, req *npmv1.NpmCommandRequest) (*npmv1.NpmCommandResponse, error) {
	return runGenericNpm(req, "install"), nil
}

func (s *NpmServer) RunScript(_ context.Context, req *npmv1.RunScriptRequest) (*npmv1.NpmCommandResponse, error) {
	if strings.TrimSpace(req.GetScript()) == "" {
		return nil, status.Error(codes.InvalidArgument, "script is required")
	}

	args := []string{"run", req.GetScript()}
	if len(req.GetArgs()) > 0 {
		args = append(args, "--")
		args = append(args, req.GetArgs()...)
	}

	result := npmrunner.RunNpm(args, defaultWorkdir(req.GetWorkdir()), req.GetEnv(), int(req.GetTimeoutS()))
	return mapCommandResponse(result), nil
}

func (s *NpmServer) Test(_ context.Context, req *npmv1.NpmCommandRequest) (*npmv1.NpmCommandResponse, error) {
	args := []string{"test"}
	if len(req.GetArgs()) > 0 {
		args = append(args, "--")
		args = append(args, req.GetArgs()...)
	}
	result := npmrunner.RunNpm(args, defaultWorkdir(req.GetWorkdir()), req.GetEnv(), int(req.GetTimeoutS()))
	return mapCommandResponse(result), nil
}

func (s *NpmServer) Build(_ context.Context, req *npmv1.NpmCommandRequest) (*npmv1.NpmCommandResponse, error) {
	args := []string{"run", "build"}
	if len(req.GetArgs()) > 0 {
		args = append(args, "--")
		args = append(args, req.GetArgs()...)
	}
	result := npmrunner.RunNpm(args, defaultWorkdir(req.GetWorkdir()), req.GetEnv(), int(req.GetTimeoutS()))
	return mapCommandResponse(result), nil
}

func (s *NpmServer) NodeExec(_ context.Context, req *npmv1.NodeExecRequest) (*npmv1.NpmCommandResponse, error) {
	if strings.TrimSpace(req.GetEval()) != "" {
		result := npmrunner.RunNodeEval(req.GetEval(), defaultWorkdir(req.GetWorkdir()), req.GetEnv(), int(req.GetTimeoutS()))
		return mapCommandResponse(result), nil
	}

	if strings.TrimSpace(req.GetFile()) == "" {
		return nil, status.Error(codes.InvalidArgument, "file or eval is required")
	}

	args := []string{req.GetFile()}
	args = append(args, req.GetArgs()...)
	result := npmrunner.RunNode(args, defaultWorkdir(req.GetWorkdir()), req.GetEnv(), int(req.GetTimeoutS()))
	return mapCommandResponse(result), nil
}

func (s *NpmServer) ListDeps(_ context.Context, req *npmv1.ListDepsRequest) (*npmv1.ListDepsResponse, error) {
	args := []string{"ls"}
	if req.GetAll() {
		args = append(args, "--all")
	}
	if req.GetDepth() > 0 {
		args = append(args, fmt.Sprintf("--depth=%d", req.GetDepth()))
	} else if !req.GetAll() {
		args = append(args, "--depth=0")
	}

	result, raw := npmrunner.RunNpmJSON(args, defaultWorkdir(req.GetWorkdir()), req.GetEnv(), 0)

	dependencies := []*npmv1.Dependency{}
	if strings.TrimSpace(string(raw)) != "" {
		var root dependencyNode
		if err := json.Unmarshal(raw, &root); err != nil {
			return nil, status.Errorf(codes.Internal, "parse npm ls json: %v", err)
		}
		dependencies = mapDependencyMap(root.Dependencies)
	}

	return &npmv1.ListDepsResponse{
		ExitCode:     int32(result.ExitCode),
		Dependencies: dependencies,
	}, nil
}

func (s *NpmServer) Audit(_ context.Context, req *npmv1.NpmCommandRequest) (*npmv1.AuditResponse, error) {
	args := []string{"audit"}
	args = append(args, req.GetArgs()...)

	result, raw := npmrunner.RunNpmJSON(args, defaultWorkdir(req.GetWorkdir()), req.GetEnv(), int(req.GetTimeoutS()))
	parsed, err := parseAudit(raw)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "parse npm audit json: %v", err)
	}
	parsed.ExitCode = int32(result.ExitCode)
	return parsed, nil
}

func (s *NpmServer) Outdated(_ context.Context, req *npmv1.NpmCommandRequest) (*npmv1.OutdatedResponse, error) {
	args := []string{"outdated"}
	args = append(args, req.GetArgs()...)

	result, raw := npmrunner.RunNpmJSON(args, defaultWorkdir(req.GetWorkdir()), req.GetEnv(), int(req.GetTimeoutS()))
	packages, err := parseOutdated(raw)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "parse npm outdated json: %v", err)
	}

	return &npmv1.OutdatedResponse{
		ExitCode: int32(result.ExitCode),
		Packages: packages,
	}, nil
}

func (s *NpmServer) ReadPackageJson(_ context.Context, req *npmv1.ReadPackageJsonRequest) (*npmv1.PackageJsonResponse, error) {
	pkg, err := pkgjson.Read(defaultWorkdir(req.GetWorkdir()))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, status.Errorf(codes.NotFound, "package.json not found: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "read package.json: %v", err)
	}

	return &npmv1.PackageJsonResponse{
		Name:             pkg.Name,
		Version:          pkg.Version,
		Description:      pkg.Description,
		Main:             pkg.Main,
		License:          pkg.License,
		Scripts:          pkg.Scripts,
		Dependencies:     pkg.Dependencies,
		DevDependencies:  pkg.DevDependencies,
		PeerDependencies: pkg.PeerDependencies,
		RawJson:          pkg.Raw,
	}, nil
}

func (s *NpmServer) Pack(_ context.Context, req *npmv1.NpmCommandRequest) (*npmv1.NpmCommandResponse, error) {
	return runGenericNpm(req, "pack"), nil
}

func (s *NpmServer) Publish(_ context.Context, req *npmv1.NpmCommandRequest) (*npmv1.NpmCommandResponse, error) {
	return runGenericNpm(req, "publish"), nil
}

func (s *NpmServer) CacheClean(_ context.Context, req *npmv1.NpmCommandRequest) (*npmv1.NpmCommandResponse, error) {
	return runGenericNpm(req, "cache", "clean"), nil
}

func (s *NpmServer) Npx(_ context.Context, req *npmv1.NpmCommandRequest) (*npmv1.NpmCommandResponse, error) {
	return runGenericNpm(req, "exec"), nil
}

func (s *NpmServer) Version(_ context.Context, _ *npmv1.VersionRequest) (*npmv1.VersionResponse, error) {
	npmResult := npmrunner.RunNpm([]string{"--version"}, ".", nil, 10)
	if npmResult.ExitCode != 0 {
		return nil, status.Errorf(codes.Internal, "npm --version failed: %s", strings.TrimSpace(npmResult.Stderr))
	}

	nodeResult := npmrunner.RunNode([]string{"--version"}, ".", nil, 10)
	if nodeResult.ExitCode != 0 {
		return nil, status.Errorf(codes.Internal, "node --version failed: %s", strings.TrimSpace(nodeResult.Stderr))
	}

	npxResult := npmrunner.RunNpm([]string{"exec", "--version"}, ".", nil, 10)
	npxVersion := strings.TrimSpace(npxResult.Stdout)
	if npxVersion == "" {
		npxVersion = strings.TrimSpace(npmResult.Stdout)
	}

	return &npmv1.VersionResponse{
		NpmVersion:  strings.TrimSpace(npmResult.Stdout),
		NodeVersion: strings.TrimSpace(nodeResult.Stdout),
		NpxVersion:  npxVersion,
	}, nil
}

func runGenericNpm(req *npmv1.NpmCommandRequest, prefix ...string) *npmv1.NpmCommandResponse {
	args := make([]string, 0, len(prefix)+len(req.GetArgs()))
	args = append(args, prefix...)
	args = append(args, req.GetArgs()...)
	result := npmrunner.RunNpm(args, defaultWorkdir(req.GetWorkdir()), req.GetEnv(), int(req.GetTimeoutS()))
	return mapCommandResponse(result)
}

func mapCommandResponse(result npmrunner.Result) *npmv1.NpmCommandResponse {
	return &npmv1.NpmCommandResponse{
		ExitCode: int32(result.ExitCode),
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
		ElapsedS: float32(result.Elapsed),
	}
}

func defaultWorkdir(workdir string) string {
	if workdir == "" {
		return "."
	}
	return workdir
}

type dependencyNode struct {
	Name         string                     `json:"name"`
	Version      string                     `json:"version"`
	Resolved     string                     `json:"resolved"`
	Dev          bool                       `json:"dev"`
	Optional     bool                       `json:"optional"`
	Problems     []string                   `json:"problems"`
	Dependencies map[string]*dependencyNode `json:"dependencies"`
}

func mapDependencyMap(in map[string]*dependencyNode) []*npmv1.Dependency {
	if len(in) == 0 {
		return []*npmv1.Dependency{}
	}
	keys := make([]string, 0, len(in))
	for k := range in {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make([]*npmv1.Dependency, 0, len(keys))
	for _, name := range keys {
		node := in[name]
		depName := name
		if node != nil && strings.TrimSpace(node.Name) != "" {
			depName = node.Name
		}

		dep := &npmv1.Dependency{Name: depName}
		if node != nil {
			dep.Version = node.Version
			dep.Resolved = node.Resolved
			dep.Dev = node.Dev
			dep.Optional = node.Optional
			dep.Problems = append([]string{}, node.Problems...)
			dep.Dependencies = mapDependencyMap(node.Dependencies)
		}
		out = append(out, dep)
	}
	return out
}

type auditReport struct {
	Metadata struct {
		Vulnerabilities struct {
			Info     int32 `json:"info"`
			Low      int32 `json:"low"`
			Moderate int32 `json:"moderate"`
			High     int32 `json:"high"`
			Critical int32 `json:"critical"`
			Total    int32 `json:"total"`
		} `json:"vulnerabilities"`
	} `json:"metadata"`
	Vulnerabilities map[string]auditVulnerability `json:"vulnerabilities"`
	Advisories      map[string]auditAdvisory      `json:"advisories"`
}

type auditVulnerability struct {
	Name         string `json:"name"`
	Severity     string `json:"severity"`
	Title        string `json:"title"`
	URL          string `json:"url"`
	Range        string `json:"range"`
	Via          []any  `json:"via"`
	FixAvailable any    `json:"fixAvailable"`
}

type auditAdvisory struct {
	ModuleName         string `json:"module_name"`
	Severity           string `json:"severity"`
	Title              string `json:"title"`
	URL                string `json:"url"`
	VulnerableVersions string `json:"vulnerable_versions"`
}

func parseAudit(raw []byte) (*npmv1.AuditResponse, error) {
	resp := &npmv1.AuditResponse{
		Vulnerabilities: []*npmv1.Vulnerability{},
		RawJson:         string(raw),
	}

	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return resp, nil
	}

	var report auditReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return nil, err
	}

	resp.Info = report.Metadata.Vulnerabilities.Info
	resp.Low = report.Metadata.Vulnerabilities.Low
	resp.Moderate = report.Metadata.Vulnerabilities.Moderate
	resp.High = report.Metadata.Vulnerabilities.High
	resp.Critical = report.Metadata.Vulnerabilities.Critical
	resp.TotalVulnerabilities = report.Metadata.Vulnerabilities.Total

	if len(report.Vulnerabilities) > 0 {
		keys := make([]string, 0, len(report.Vulnerabilities))
		for k := range report.Vulnerabilities {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, key := range keys {
			v := report.Vulnerabilities[key]
			name := v.Name
			if strings.TrimSpace(name) == "" {
				name = key
			}
			title := v.Title
			url := v.URL
			via := stringifyVia(v.Via, &title, &url)
			resp.Vulnerabilities = append(resp.Vulnerabilities, &npmv1.Vulnerability{
				Name:         name,
				Severity:     v.Severity,
				Title:        title,
				Url:          url,
				Range:        v.Range,
				Via:          via,
				FixAvailable: stringifyFix(v.FixAvailable),
			})
		}
	}

	if len(resp.Vulnerabilities) == 0 && len(report.Advisories) > 0 {
		keys := make([]string, 0, len(report.Advisories))
		for k := range report.Advisories {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, key := range keys {
			a := report.Advisories[key]
			name := a.ModuleName
			if strings.TrimSpace(name) == "" {
				name = key
			}
			resp.Vulnerabilities = append(resp.Vulnerabilities, &npmv1.Vulnerability{
				Name:     name,
				Severity: a.Severity,
				Title:    a.Title,
				Url:      a.URL,
				Range:    a.VulnerableVersions,
			})
		}
	}

	if resp.TotalVulnerabilities == 0 && len(resp.Vulnerabilities) > 0 {
		resp.TotalVulnerabilities = int32(len(resp.Vulnerabilities))
	}

	return resp, nil
}

func stringifyVia(via []any, title *string, url *string) []string {
	if len(via) == 0 {
		return []string{}
	}

	out := make([]string, 0, len(via))
	for _, v := range via {
		switch vv := v.(type) {
		case string:
			out = append(out, vv)
		case map[string]any:
			if name, ok := vv["name"].(string); ok && name != "" {
				out = append(out, name)
			}
			if *title == "" {
				if t, ok := vv["title"].(string); ok {
					*title = t
				}
			}
			if *url == "" {
				if u, ok := vv["url"].(string); ok {
					*url = u
				}
			}
		default:
			out = append(out, fmt.Sprintf("%v", vv))
		}
	}
	return out
}

func stringifyFix(in any) []string {
	switch v := in.(type) {
	case nil:
		return []string{}
	case bool:
		if v {
			return []string{"true"}
		}
		return []string{"false"}
	case string:
		if strings.TrimSpace(v) == "" {
			return []string{}
		}
		return []string{v}
	case map[string]any:
		parts := make([]string, 0, len(v))
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%s=%v", k, v[k]))
		}
		return parts
	default:
		return []string{fmt.Sprintf("%v", v)}
	}
}

type outdatedEntry struct {
	Current  string `json:"current"`
	Wanted   string `json:"wanted"`
	Latest   string `json:"latest"`
	Location string `json:"location"`
	Type     string `json:"type"`
}

func parseOutdated(raw []byte) ([]*npmv1.OutdatedPackage, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return []*npmv1.OutdatedPackage{}, nil
	}

	var entries map[string]outdatedEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return []*npmv1.OutdatedPackage{}, nil
	}

	keys := make([]string, 0, len(entries))
	for k := range entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make([]*npmv1.OutdatedPackage, 0, len(keys))
	for _, name := range keys {
		entry := entries[name]
		out = append(out, &npmv1.OutdatedPackage{
			Name:     name,
			Current:  entry.Current,
			Wanted:   entry.Wanted,
			Latest:   entry.Latest,
			Location: entry.Location,
			Type:     entry.Type,
		})
	}
	return out, nil
}
