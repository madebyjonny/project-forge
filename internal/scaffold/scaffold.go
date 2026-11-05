// Package scaffold owns the embedded project templates, features and stubs.
package scaffold

import (
	"bytes"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed all:templates
var templates embed.FS

// Version is stamped into forge.json of generated projects.
var Version = "0.1.0"

// Dependency versions for generated projects. Bump here, every new project follows.
var versions = map[string]string{
	"hono":          "^4.13.16",
	"drizzle-orm":   "^0.45.4",
	"zod":           "^4.6.7",
	"pg":            "^8.23.1",
	"better-auth":   "^1.7.18",
	"@casl/ability": "^7.1.1",
	"bullmq":        "^6.7.0",
	"ioredis":       "^6.0.0",
	"drizzle-kit":   "^0.31.11",
	"@types/bun":    "^1.4.4",
	"@types/pg":     "^8.23.1",
	"typescript":    "^5.9.3",
}

func deps(names ...string) map[string]string {
	out := map[string]string{}
	for _, n := range names {
		out[n] = versions[n]
	}
	return out
}

// copyTree copies templates/<dir> into dest. Existing files are skipped unless force.
func copyTree(dir, dest string, force bool) ([]string, error) {
	root := path.Join("templates", dir)
	var written []string
	err := fs.WalkDir(templates, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(strings.TrimPrefix(p, root), "/")
		if d.IsDir() || rel == "" {
			return nil
		}
		target := filepath.Join(dest, filepath.FromSlash(rel))
		if _, err := os.Stat(target); err == nil && !force {
			return nil
		}
		data, err := templates.ReadFile(p)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		written = append(written, rel)
		return os.WriteFile(target, data, 0o644)
	})
	return written, err
}

// Stub renders a generator stub. A project can override any stub by placing a
// copy at stubs/<name>.tmpl (see `forge stub:publish`).
func Stub(projectRoot, name string, data any) ([]byte, error) {
	src, err := StubSource(projectRoot, name)
	if err != nil {
		return nil, err
	}
	tpl, err := template.New(name).Parse(string(src))
	if err != nil {
		return nil, fmt.Errorf("stub %s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("stub %s: %w", name, err)
	}
	return buf.Bytes(), nil
}

func StubSource(projectRoot, name string) ([]byte, error) {
	if projectRoot != "" {
		if data, err := os.ReadFile(filepath.Join(projectRoot, "stubs", name+".tmpl")); err == nil {
			return data, nil
		}
	}
	return templates.ReadFile(path.Join("templates/stubs/default", name+".tmpl"))
}

// StubNames lists the built-in stubs.
func StubNames() []string {
	entries, _ := templates.ReadDir("templates/stubs/default")
	var names []string
	for _, e := range entries {
		names = append(names, strings.TrimSuffix(e.Name(), ".tmpl"))
	}
	return names
}

// Secret returns a random URL-safe key suitable for APP_KEY / AUTH_SECRET.
func Secret() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return "base64:" + base64.RawURLEncoding.EncodeToString(b)
}
