package progress

// A simple example that shows how to render a progress bar in a "pure"
// fashion. In this example we bump the progress by 25% every second,
// maintaining the progress state on our top level model using the progress bar
// model's ViewAs method only for rendering.
//
// The signature for ViewAs is:
//
//     func (m Model) ViewAs(percent float64) string
//
// So it takes a float between 0 and 1, and renders the progress bar
// accordingly. When using the progress bar in this "pure" fashion and there's
// no need to call an Update method.
//
// The progress bar is also able to animate itself, however. For details see
// the progress-animated example.

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	padding  = 2
	maxWidth = 80
)

func newModel() *model {
	return &model{progress: progress.New(
		// progress.WithSolidFill("cyan"),
		progress.WithScaledGradient("#FF7CCB", "#FDFF8C"),
		progress.WithoutPercentage(),
	)}
	// return &model{progress: progress.New(progress.WithScaledGradient("#FF7CCB", "#FDFF8C"))}
}

func Get() *model {
	if program != nil {
		return program
	}
	return newModel()
}

func Run() {
	if _, err := tea.NewProgram(newModel()).Run(); err != nil {
		fmt.Println("Oh no!", err)
		os.Exit(1)
	}
}

var program *model

type model struct {
	percent  float64
	progress progress.Model
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - padding*2 - 4
		if m.progress.Width > maxWidth {
			m.progress.Width = maxWidth
		}

	case Percent:
		m.percent = min(max(0.0, float64(msg)), 1.0)
	}
	return m, nil
}

func (m model) View() string {
	pad := strings.Repeat(" ", padding)
	return "\n" +
		pad + m.progress.ViewAs(m.percent) + "\n\n"
}

func (m model) ViewAs(percent float64) string {
	return m.progress.ViewAs(percent)
}

type Percent float64
