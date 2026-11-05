package runner

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Tunnel exposes a local port on a public HTTPS URL (needed by hosted MCP
// clients such as ChatGPT, which can't reach localhost).
type Tunnel struct {
	Provider string
	URL      string
	cmd      *exec.Cmd
}

var TunnelProviders = []string{"ngrok", "cloudflared"}

var cloudflaredURL = regexp.MustCompile(`https://[a-z0-9-]+\.trycloudflare\.com`)

// StartTunnel starts provider ("" picks the first one installed) and waits for its public URL.
func StartTunnel(provider string, port int, domain string) (*Tunnel, error) {
	if provider == "" || provider == "auto" {
		for _, p := range TunnelProviders {
			if _, err := exec.LookPath(p); err == nil {
				provider = p
				break
			}
		}
		if provider == "" || provider == "auto" {
			return nil, fmt.Errorf("no tunnel tool found — install ngrok (https://ngrok.com/download) or cloudflared (brew install cloudflared)")
		}
	}
	if _, err := exec.LookPath(provider); err != nil {
		return nil, fmt.Errorf("%s is not installed", provider)
	}

	var cmd *exec.Cmd
	var out io.ReadCloser
	var err error
	switch provider {
	case "ngrok":
		args := []string{"http", strconv.Itoa(port), "--log", "stdout", "--log-format", "json"}
		if domain != "" {
			args = append(args, "--url", "https://"+strings.TrimPrefix(strings.TrimPrefix(domain, "https://"), "http://"))
		}
		cmd = exec.Command("ngrok", args...)
		out, err = cmd.StdoutPipe()
	case "cloudflared":
		if domain != "" {
			return nil, fmt.Errorf("--domain needs a named cloudflared tunnel; quick tunnels get a random URL")
		}
		cmd = exec.Command("cloudflared", "tunnel", "--no-autoupdate", "--url", "http://localhost:"+strconv.Itoa(port))
		out, err = cmd.StderrPipe()
	default:
		return nil, fmt.Errorf("unknown tunnel %q (%s)", provider, strings.Join(TunnelProviders, ", "))
	}
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	found := make(chan string, 1)
	failed := make(chan error, 1)
	// Non-blocking sends: the goroutine must keep draining output for the tunnel's lifetime.
	report := func(u string) {
		select {
		case found <- u:
		default:
		}
	}
	fail := func(err error) {
		select {
		case failed <- err:
		default:
		}
	}
	go func() {
		scanner := bufio.NewScanner(out)
		for scanner.Scan() {
			line := scanner.Text()
			if provider == "cloudflared" {
				if u := cloudflaredURL.FindString(line); u != "" {
					report(u)
				}
				continue
			}
			var rec map[string]any
			if json.Unmarshal([]byte(line), &rec) != nil {
				continue
			}
			if u, _ := rec["url"].(string); strings.HasPrefix(u, "https://") && rec["msg"] == "started tunnel" {
				report(u)
			}
			if lvl, _ := rec["lvl"].(string); lvl == "eror" || lvl == "crit" {
				msg, _ := rec["err"].(string)
				if msg == "" {
					msg, _ = rec["msg"].(string)
				}
				fail(fmt.Errorf("ngrok: %s", firstLine(msg)))
			}
		}
		fail(fmt.Errorf("%s exited before reporting a URL (is it logged in / configured?)", provider))
	}()

	t := &Tunnel{Provider: provider, cmd: cmd}
	select {
	case t.URL = <-found:
		return t, nil
	case err := <-failed:
		t.Stop()
		return nil, err
	case <-time.After(30 * time.Second):
		t.Stop()
		return nil, fmt.Errorf("timed out waiting for %s to report a public URL", provider)
	}
}

func (t *Tunnel) Stop() {
	if t != nil && t.cmd != nil && t.cmd.Process != nil {
		_ = t.cmd.Process.Kill()
		_ = t.cmd.Wait()
	}
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return line
}
