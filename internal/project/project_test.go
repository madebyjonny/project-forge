package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnvFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	os.WriteFile(path, []byte("# app\nAPP_NAME=demo\nDB_CONNECTION=sqlite\n"), 0o644)
	e, err := OpenEnv(path)
	if err != nil {
		t.Fatal(err)
	}
	e.Set("DB_CONNECTION", "postgres")
	e.Default("APP_NAME", "ignored")
	e.Section("Queue")
	e.Set("REDIS_URL", "redis://x:6379")
	e.Set("QUOTED", "has space")
	if err := e.Save(); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	want := "# app\nAPP_NAME=demo\nDB_CONNECTION=postgres\n\n# Queue\nREDIS_URL=redis://x:6379\nQUOTED=\"has space\"\n"
	if string(data) != want {
		t.Errorf("got:\n%s\nwant:\n%s", data, want)
	}
	if v, _ := e.Get("QUOTED"); v != "has space" {
		t.Errorf("Get = %q", v)
	}
}

func TestInsertBefore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "routes.ts")
	os.WriteFile(path, []byte("export default routes((r) => {\n  // forge:routes\n});\n"), 0o644)
	for range 2 { // idempotent
		if err := InsertBefore(path, "routes", `r.resource("/posts", PostController);`); err != nil {
			t.Fatal(err)
		}
	}
	data, _ := os.ReadFile(path)
	if strings.Count(string(data), "r.resource") != 1 || !strings.Contains(string(data), "  r.resource(\"/posts\", PostController);\n  // forge:routes") {
		t.Errorf("unexpected content:\n%s", data)
	}
	if err := InsertBefore(path, "routes", "SendPost,"); err != nil {
		t.Fatal(err)
	}
	if err := InsertBefore(path, "routes", "Post,"); err != nil || !strings.Contains(readFile(path), "\n  Post,\n") {
		t.Errorf("substring of an existing line must still be inserted: %v", err)
	}
	if err := InsertBefore(path, "missing", "x"); err == nil {
		t.Error("expected missing marker error")
	}
}

func TestPackageJSONOrder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "package.json")
	pkg := NewPackageJSON(path, map[string]any{"type": "module", "name": "x", "zzz": 1})
	pkg.AddDeps(map[string]string{"hono": "^4"}, false)
	pkg.SetScripts(map[string]string{"dev": "bun --hot server.ts"})
	if err := pkg.Save(); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	s := string(data)
	order := []string{`"name"`, `"type"`, `"scripts"`, `"dependencies"`, `"zzz"`}
	last := -1
	for _, k := range order {
		i := strings.Index(s, k)
		if i < last {
			t.Fatalf("key %s out of order:\n%s", k, s)
		}
		last = i
	}
}

func readFile(path string) string {
	data, _ := os.ReadFile(path)
	return string(data)
}
