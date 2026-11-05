package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteCodexConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CODEX_HOME", home)
	path := filepath.Join(home, "config.toml")
	os.WriteFile(path, []byte("model = \"x\"\n\n[mcp_servers.other]\ncommand = \"npx\"\n"), 0o644)

	if _, err := writeCodexConfig("notes", "http://localhost:3000/mcp"); err != nil {
		t.Fatal(err)
	}
	if _, err := writeCodexConfig("notes", "https://notes.example.com/mcp"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	got := string(data)
	want := "model = \"x\"\n\n[mcp_servers.other]\ncommand = \"npx\"\n\n[mcp_servers.notes]\nurl = \"https://notes.example.com/mcp\"\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	if strings.Count(got, "[mcp_servers.notes]") != 1 {
		t.Error("section duplicated")
	}
}
