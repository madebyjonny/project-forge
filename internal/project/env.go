package project

import (
	"os"
	"strings"
)

// EnvFile is a minimal .env editor that preserves comments and ordering.
type EnvFile struct {
	path  string
	lines []string
}

func OpenEnv(path string) (*EnvFile, error) {
	e := &EnvFile{path: path}
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if len(data) > 0 {
		e.lines = strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	}
	return e, nil
}

func (e *EnvFile) index(key string) int {
	for i, l := range e.lines {
		if k, _, ok := strings.Cut(strings.TrimSpace(l), "="); ok && strings.TrimSpace(k) == key {
			return i
		}
	}
	return -1
}

func (e *EnvFile) Get(key string) (string, bool) {
	if i := e.index(key); i >= 0 {
		_, v, _ := strings.Cut(e.lines[i], "=")
		return strings.Trim(strings.TrimSpace(v), `"'`), true
	}
	return "", false
}

// Set replaces an existing key or appends it.
func (e *EnvFile) Set(key, value string) {
	line := key + "=" + quote(value)
	if i := e.index(key); i >= 0 {
		e.lines[i] = line
		return
	}
	e.lines = append(e.lines, line)
}

// Default sets a key only if it's missing.
func (e *EnvFile) Default(key, value string) {
	if e.index(key) < 0 {
		e.Set(key, value)
	}
}

// Section starts a commented block (once) so appended keys stay grouped.
func (e *EnvFile) Section(comment string) {
	for _, l := range e.lines {
		if l == "# "+comment {
			return
		}
	}
	if len(e.lines) > 0 && e.lines[len(e.lines)-1] != "" {
		e.lines = append(e.lines, "")
	}
	e.lines = append(e.lines, "# "+comment)
}

func (e *EnvFile) Save() error {
	return os.WriteFile(e.path, []byte(strings.Join(e.lines, "\n")+"\n"), 0o644)
}

func quote(v string) string {
	if strings.ContainsAny(v, " #\"'") {
		return `"` + strings.ReplaceAll(v, `"`, `\"`) + `"`
	}
	return v
}
