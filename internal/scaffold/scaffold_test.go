package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/madebyjonny/forge/internal/project"
)

func TestCreateAndInstall(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	p, err := Create(Options{Name: "demo", Dir: dir, Database: "sqlite", Features: []string{"auth"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"server.ts", "core/app.ts", "core/auth.ts", "app/models/user.ts", "routes/web.ts", ".env", "forge.json"} {
		if !p.Exists(f) {
			t.Errorf("missing %s", f)
		}
	}

	// Installing twice must not duplicate registrations.
	f, _ := GetFeature("realtime")
	for range 2 {
		if err := Install(p, f); err != nil {
			t.Fatal(err)
		}
	}
	reloaded, err := project.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reloaded.Has("auth") || !reloaded.Has("realtime") || len(reloaded.Manifest.Features) != 2 {
		t.Errorf("features = %v", reloaded.Manifest.Features)
	}
	boot, _ := os.ReadFile(filepath.Join(dir, "bootstrap/features.ts"))
	if !strings.Contains(string(boot), "[auth(), policies(), realtime()]") {
		t.Errorf("bootstrap:\n%s", boot)
	}
	models, _ := os.ReadFile(filepath.Join(dir, "app/models/index.ts"))
	if strings.Count(string(models), `export * from "./user";`) != 1 {
		t.Errorf("models index:\n%s", models)
	}
	env, _ := project.OpenEnv(filepath.Join(dir, ".env"))
	if v, _ := env.Get("AUTH_SECRET"); !strings.HasPrefix(v, "base64:") {
		t.Errorf("AUTH_SECRET = %q", v)
	}
	example, _ := project.OpenEnv(filepath.Join(dir, ".env.example"))
	if v, _ := example.Get("APP_KEY"); v != "" {
		t.Error(".env.example must not contain secrets")
	}
}

func TestCreateRefusesNonEmptyDir(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "keep"), []byte("x"), 0o644)
	if _, err := Create(Options{Name: "demo", Dir: dir, Database: "sqlite"}); err == nil {
		t.Error("expected error for non-empty dir")
	}
}

func TestMCPRequiresAuth(t *testing.T) {
	features, err := Resolve([]string{"mcp"})
	if err != nil {
		t.Fatal(err)
	}
	if len(features) != 2 || features[0].Name != "auth" || features[1].Name != "mcp" {
		t.Fatalf("Resolve(mcp) = %v", features)
	}

	dir := filepath.Join(t.TempDir(), "demo")
	p, err := Create(Options{Name: "demo", Dir: dir, Database: "sqlite", Features: []string{"mcp"}})
	if err != nil {
		t.Fatal(err)
	}
	if !p.Has("auth") || !p.Has("mcp") {
		t.Fatalf("features = %v", p.Manifest.Features)
	}
	authBoot, _ := os.ReadFile(filepath.Join(dir, "bootstrap/auth.ts"))
	if !strings.Contains(string(authBoot), "[mcpAuth()]") {
		t.Errorf("bootstrap/auth.ts:\n%s", authBoot)
	}
	models, _ := os.ReadFile(filepath.Join(dir, "app/models/index.ts"))
	if !strings.Contains(string(models), `export * from "./oauth";`) {
		t.Errorf("models index:\n%s", models)
	}

	bare := &project.Project{Root: t.TempDir(), Manifest: project.Manifest{}}
	mcp, _ := GetFeature("mcp")
	if err := Install(bare, mcp); err == nil || !strings.Contains(err.Error(), "requires the auth feature") {
		t.Errorf("expected requirement error, got %v", err)
	}
}
