// Package project locates and edits a forge project on disk.
package project

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const ManifestFile = "forge.json"

// Manifest is forge.json — the contract between the CLI and a project.
// Stack names the implementation behind each capability; commands stay the same
// no matter which implementation is installed.
type Manifest struct {
	Name     string            `json:"name"`
	Forge    string            `json:"forge"`
	Runtime  string            `json:"runtime"`
	Stack    map[string]string `json:"stack"`
	Features []string          `json:"features"`
}

type Project struct {
	Root     string
	Manifest Manifest
}

var ErrNotProject = errors.New("not inside a forge project (no forge.json found) — run `forge new <name>` first")

// Find walks up from dir looking for forge.json.
func Find(dir string) (*Project, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ManifestFile)); err == nil {
			return Load(dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, ErrNotProject
		}
		dir = parent
	}
}

func Current() (*Project, error) { return Find(".") }

func Load(root string) (*Project, error) {
	data, err := os.ReadFile(filepath.Join(root, ManifestFile))
	if err != nil {
		return nil, err
	}
	p := &Project{Root: root}
	if err := json.Unmarshal(data, &p.Manifest); err != nil {
		return nil, fmt.Errorf("invalid %s: %w", ManifestFile, err)
	}
	if p.Manifest.Runtime == "" {
		p.Manifest.Runtime = "bun"
	}
	return p, nil
}

func (p *Project) Save() error {
	slices.Sort(p.Manifest.Features)
	p.Manifest.Features = slices.Compact(p.Manifest.Features)
	return WriteJSON(filepath.Join(p.Root, ManifestFile), p.Manifest)
}

func (p *Project) Path(parts ...string) string {
	return filepath.Join(append([]string{p.Root}, parts...)...)
}

func (p *Project) Has(feature string) bool { return slices.Contains(p.Manifest.Features, feature) }

func (p *Project) Exists(rel string) bool {
	_, err := os.Stat(p.Path(rel))
	return err == nil
}

// Stack returns the implementation configured for a capability, e.g. Stack("orm") == "drizzle".
func (p *Project) Stack(capability string) string { return p.Manifest.Stack[capability] }

func WriteJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// WriteFile writes a file, creating parent dirs. It refuses to overwrite unless force is set.
func WriteFile(path string, content []byte, force bool) error {
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists (use --force to overwrite)", path)
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}

// InsertBefore adds line above the `// forge:<marker>` comment in a registry file,
// matching its indentation. It's idempotent.
func InsertBefore(path, marker, line string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	for _, l := range lines {
		if strings.TrimSpace(l) == strings.TrimSpace(line) {
			return nil
		}
	}
	tag := "// forge:" + marker
	for i, l := range lines {
		if strings.TrimSpace(l) == tag {
			indent := l[:len(l)-len(strings.TrimLeft(l, " \t"))]
			lines = slices.Insert(lines, i, indent+line)
			return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
		}
	}
	return fmt.Errorf("%s: marker %q not found — add it back so forge can register things here", path, tag)
}
