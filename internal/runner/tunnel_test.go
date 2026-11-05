package runner

import (
	"os"
	"path/filepath"
	"testing"
)

func fakeBin(t *testing.T, name, script string) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestNgrokTunnelURL(t *testing.T) {
	fakeBin(t, "ngrok", `echo '{"lvl":"info","msg":"started tunnel","url":"https://demo.ngrok.app"}'
exec sleep 30
`)
	tun, err := StartTunnel("ngrok", 3000, "")
	if err != nil {
		t.Fatal(err)
	}
	defer tun.Stop()
	if tun.URL != "https://demo.ngrok.app" {
		t.Errorf("URL = %q", tun.URL)
	}
}

func TestNgrokTunnelError(t *testing.T) {
	fakeBin(t, "ngrok", `printf '%s\n' '{"lvl":"eror","msg":"command failed","err":"authentication failed: bad token\nmore"}'
`)
	if _, err := StartTunnel("ngrok", 3000, ""); err == nil || err.Error() != "ngrok: authentication failed: bad token" {
		t.Errorf("err = %v", err)
	}
}

func TestCloudflaredTunnelURL(t *testing.T) {
	fakeBin(t, "cloudflared", `echo 'INF |  https://quick-fox-test.trycloudflare.com  |' >&2
exec sleep 30
`)
	tun, err := StartTunnel("cloudflared", 3000, "")
	if err != nil {
		t.Fatal(err)
	}
	defer tun.Stop()
	if tun.URL != "https://quick-fox-test.trycloudflare.com" {
		t.Errorf("URL = %q", tun.URL)
	}
}
