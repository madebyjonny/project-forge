package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// Step is one unit of work shown in a task list. Run must not write to stdout.
type Step struct {
	Title string
	Run   func() error
}

type stepDone struct {
	err error
	dur time.Duration
}

type tasksModel struct {
	steps   []Step
	current int
	durs    []time.Duration
	err     error
	spin    spinner.Model
}

func (m tasksModel) run() tea.Cmd {
	step := m.steps[m.current]
	return func() tea.Msg {
		start := time.Now()
		err := step.Run()
		return stepDone{err: err, dur: time.Since(start)}
	}
}

func (m tasksModel) Init() tea.Cmd { return tea.Batch(m.spin.Tick, m.run()) }

func (m tasksModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.err = fmt.Errorf("cancelled")
			return m, tea.Quit
		}
	case stepDone:
		m.durs = append(m.durs, msg.dur)
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.current++
		if m.current >= len(m.steps) {
			return m, tea.Quit
		}
		return m, m.run()
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m tasksModel) View() string {
	var b strings.Builder
	for i, s := range m.steps {
		switch {
		case i < m.current:
			fmt.Fprintf(&b, "  %s %s %s\n", OK.Render("✓"), s.Title, Faint.Render(fmtDur(m.durs[i])))
		case i == m.current && m.err != nil:
			fmt.Fprintf(&b, "  %s %s\n", Bad.Render("✗"), s.Title)
		case i == m.current:
			fmt.Fprintf(&b, "  %s %s\n", m.spin.View(), s.Title)
		default:
			fmt.Fprintf(&b, "  %s %s\n", Faint.Render("·"), Faint.Render(s.Title))
		}
	}
	return b.String()
}

func fmtDur(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

// RunSteps executes steps in order with a live spinner list, or plain lines
// when not attached to a terminal. Stops at the first error.
func RunSteps(steps []Step) error {
	if len(steps) == 0 {
		return nil
	}
	if !Interactive() {
		for _, s := range steps {
			start := time.Now()
			fmt.Printf("  • %s ", s.Title)
			if err := s.Run(); err != nil {
				fmt.Println(Bad.Render("✗"))
				return err
			}
			fmt.Println(OK.Render("✓") + " " + Faint.Render(fmtDur(time.Since(start))))
		}
		return nil
	}
	sp := spinner.New(spinner.WithSpinner(spinner.MiniDot), spinner.WithStyle(Accent))
	final, err := tea.NewProgram(tasksModel{steps: steps, spin: sp}).Run()
	if err != nil {
		return err
	}
	return final.(tasksModel).err
}
