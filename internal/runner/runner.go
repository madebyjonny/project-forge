// Package runner executes project scripts. The CLI never calls drizzle-kit,
// better-auth etc. directly — only the `forge:*` scripts in package.json — so the
// tooling behind a command can change without the command changing.
package runner

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/madebyjonny/forge/internal/project"
)

type Options struct {
	Env    string // NODE_ENV, selects .env.<env>
	Extra  []string
	Stdout io.Writer
	Stderr io.Writer
	Vars   map[string]string
}

func Runtime(p *project.Project) string {
	if p != nil && p.Manifest.Runtime != "" {
		return p.Manifest.Runtime
	}
	return "bun"
}

func EnsureRuntime(p *project.Project) error {
	rt := Runtime(p)
	if _, err := exec.LookPath(rt); err != nil {
		return fmt.Errorf("%s is not installed — install it from https://bun.sh (curl -fsSL https://bun.sh/install | bash)", rt)
	}
	return nil
}

// Script runs `bun run <script> [extra...]` in the project root.
func Script(p *project.Project, script string, o Options) error {
	pkg, err := project.OpenPackageJSON(p.Path("package.json"))
	if err != nil {
		return err
	}
	if _, ok := pkg.Script(script); !ok {
		return fmt.Errorf("package.json has no %q script — this command isn't wired up for this project", script)
	}
	return Exec(p, append([]string{"run", script}, o.Extra...), o)
}

// Exec runs the project runtime with args.
func Exec(p *project.Project, args []string, o Options) error {
	if err := EnsureRuntime(p); err != nil {
		return err
	}
	cmd := exec.Command(Runtime(p), args...)
	cmd.Dir = p.Root
	cmd.Stdin = os.Stdin
	cmd.Stdout = or(o.Stdout, os.Stdout)
	cmd.Stderr = or(o.Stderr, os.Stderr)
	cmd.Env = os.Environ()
	if o.Env != "" {
		cmd.Env = append(cmd.Env, "NODE_ENV="+o.Env)
	}
	for k, v := range o.Vars {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	// Forward Ctrl+C / SIGTERM to the child and wait for it, so forge's own
	// cleanup (e.g. closing a tunnel) still runs afterwards.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigs)
	if err := cmd.Start(); err != nil {
		return err
	}
	done := make(chan struct{})
	defer close(done)
	go func() {
		for {
			select {
			case s := <-sigs:
				_ = cmd.Process.Signal(s)
			case <-done:
				return
			}
		}
	}()
	return cmd.Wait()
}

// Capture runs a script and returns its stdout; stderr is included in errors.
func Capture(p *project.Project, script string, o Options) ([]byte, error) {
	var out, errb bytes.Buffer
	o.Stdout, o.Stderr = &out, &errb
	if err := Script(p, script, o); err != nil {
		return nil, fmt.Errorf("%w\n%s", err, bytes.TrimSpace(append(errb.Bytes(), out.Bytes()...)))
	}
	return out.Bytes(), nil
}

// Quiet runs a script, only surfacing output if it fails.
func Quiet(p *project.Project, script string, o Options) error {
	_, err := Capture(p, script, o)
	return err
}

// Install runs `bun install` quietly.
func Install(p *project.Project) error {
	var buf bytes.Buffer
	if err := Exec(p, []string{"install"}, Options{Stdout: &buf, Stderr: &buf}); err != nil {
		return fmt.Errorf("%w\n%s", err, buf.String())
	}
	return nil
}

// ExitCode extracts a child exit status so forge can propagate it.
func ExitCode(err error) int {
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			return 128 + int(ws.Signal())
		}
		return ee.ExitCode()
	}
	return 1
}

func or(w, fallback io.Writer) io.Writer {
	if w != nil {
		return w
	}
	return fallback
}
