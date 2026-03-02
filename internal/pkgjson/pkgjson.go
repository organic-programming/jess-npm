package pkgjson

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type PackageJSON struct {
	Name             string
	Version          string
	Description      string
	Main             string
	License          string
	Scripts          map[string]string
	Dependencies     map[string]string
	DevDependencies  map[string]string
	PeerDependencies map[string]string
	Raw              string
}

func Read(dir string) (*PackageJSON, error) {
	if dir == "" {
		dir = "."
	}

	path := filepath.Join(dir, "package.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var decoded struct {
		Name             string            `json:"name"`
		Version          string            `json:"version"`
		Description      string            `json:"description"`
		Main             string            `json:"main"`
		License          string            `json:"license"`
		Scripts          map[string]string `json:"scripts"`
		Dependencies     map[string]string `json:"dependencies"`
		DevDependencies  map[string]string `json:"devDependencies"`
		PeerDependencies map[string]string `json:"peerDependencies"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, err
	}

	return &PackageJSON{
		Name:             decoded.Name,
		Version:          decoded.Version,
		Description:      decoded.Description,
		Main:             decoded.Main,
		License:          decoded.License,
		Scripts:          cloneMap(decoded.Scripts),
		Dependencies:     cloneMap(decoded.Dependencies),
		DevDependencies:  cloneMap(decoded.DevDependencies),
		PeerDependencies: cloneMap(decoded.PeerDependencies),
		Raw:              string(raw),
	}, nil
}

func cloneMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
