package project

import (
	"bytes"
	"encoding/json"
	"os"
	"slices"
	"sort"
)

// PackageJSON edits package.json, keeping the conventional key order.
type PackageJSON struct {
	path string
	data map[string]any
}

var keyOrder = []string{"name", "version", "private", "type", "scripts", "dependencies", "devDependencies"}

func NewPackageJSON(path string, data map[string]any) *PackageJSON {
	return &PackageJSON{path: path, data: data}
}

func OpenPackageJSON(path string) (*PackageJSON, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	pkg := &PackageJSON{path: path, data: map[string]any{}}
	if err := json.Unmarshal(raw, &pkg.data); err != nil {
		return nil, err
	}
	return pkg, nil
}

func (p *PackageJSON) section(name string) map[string]any {
	m, ok := p.data[name].(map[string]any)
	if !ok {
		m = map[string]any{}
		p.data[name] = m
	}
	return m
}

func (p *PackageJSON) AddDeps(deps map[string]string, dev bool) {
	key := "dependencies"
	if dev {
		key = "devDependencies"
	}
	s := p.section(key)
	for name, version := range deps {
		if _, ok := s[name]; !ok {
			s[name] = version
		}
	}
}

func (p *PackageJSON) SetScripts(scripts map[string]string) {
	s := p.section("scripts")
	for k, v := range scripts {
		s[k] = v
	}
}

func (p *PackageJSON) Script(name string) (string, bool) {
	v, ok := p.section("scripts")[name].(string)
	return v, ok
}

func (p *PackageJSON) Save() error {
	keys := make([]string, 0, len(p.data))
	for k := range p.data {
		keys = append(keys, k)
	}
	sort.SliceStable(keys, func(i, j int) bool {
		a, b := slices.Index(keyOrder, keys[i]), slices.Index(keyOrder, keys[j])
		if a < 0 {
			a = len(keyOrder)
		}
		if b < 0 {
			b = len(keyOrder)
		}
		if a != b {
			return a < b
		}
		return keys[i] < keys[j]
	})

	var buf bytes.Buffer
	buf.WriteString("{\n")
	for i, k := range keys {
		val, err := marshal(p.data[k])
		if err != nil {
			return err
		}
		key, _ := json.Marshal(k)
		buf.WriteString("  " + string(key) + ": " + string(bytes.ReplaceAll(val, []byte("\n"), []byte("\n  "))))
		if i < len(keys)-1 {
			buf.WriteString(",")
		}
		buf.WriteString("\n")
	}
	buf.WriteString("}\n")
	return os.WriteFile(p.path, buf.Bytes(), 0o644)
}

func marshal(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}
