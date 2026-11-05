package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PaletteItem is a command shown in the interactive palette.
type PaletteItem struct {
	Name, Group, Desc string
}

func (i PaletteItem) FilterValue() string { return i.Name + " " + i.Desc }

type paletteDelegate struct{}

func (paletteDelegate) Height() int                         { return 1 }
func (paletteDelegate) Spacing() int                        { return 0 }
func (paletteDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }
func (paletteDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	it := item.(PaletteItem)
	name := fmt.Sprintf("%-20s", it.Name)
	if index == m.Index() {
		fmt.Fprint(w, Accent.Render("› "+name)+" "+it.Desc)
		return
	}
	fmt.Fprint(w, "  "+Code.Render(name)+" "+Faint.Render(it.Desc))
}

type paletteModel struct {
	list   list.Model
	choice string
}

func (m paletteModel) Init() tea.Cmd { return nil }

func (m paletteModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, min(msg.Height-1, 28))
	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}
		switch msg.String() {
		case "enter":
			if it, ok := m.list.SelectedItem().(PaletteItem); ok {
				m.choice = it.Name
			}
			return m, tea.Quit
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m paletteModel) View() string { return m.list.View() }

// Palette shows a filterable command list and returns the chosen command name.
func Palette(title string, items []PaletteItem) (string, error) {
	li := make([]list.Item, len(items))
	for i, it := range items {
		li[i] = it
	}
	l := list.New(li, paletteDelegate{}, 80, min(len(items)+6, 28))
	l.Title = strings.TrimSpace(title)
	l.Styles.Title = lipgloss.NewStyle().Foreground(lipgloss.Color("#1A1A1A")).Background(Brand).Bold(true).Padding(0, 1)
	l.SetShowStatusBar(false)
	l.Styles.FilterPrompt = Accent
	l.Styles.FilterCursor = Accent
	final, err := tea.NewProgram(paletteModel{list: l}).Run()
	if err != nil {
		return "", err
	}
	return final.(paletteModel).choice, nil
}
