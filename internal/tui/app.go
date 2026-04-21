package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gaio/tiger/internal/ipc"
)

// tickMsg fires every 500ms to refresh item state from the daemon.
type tickMsg struct{}

// itemsMsg carries a fresh item list from the daemon.
type itemsMsg struct {
	items []ipc.Item
	err   error
}

type appState int

const (
	stateList appState = iota
	stateForm
)

// App is the root bubbletea model.
type App struct {
	client    *ipc.Client
	tabs      []string
	activeTab int
	items     []ipc.Item
	cursor    int
	state     appState
	form      Form
	width     int
	height    int
	statusMsg string
}

var tabNames = []string{"Timers", "Alarms", "Stopwatches"}

var tabKinds = []ipc.ItemKind{ipc.KindTimer, ipc.KindAlarm, ipc.KindStopwatch}

func NewApp(client *ipc.Client) *App {
	return &App{
		client: client,
		tabs:   tabNames,
	}
}

func (a *App) Init() tea.Cmd {
	return tea.Batch(fetchItems(a.client), tick())
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height

	case tickMsg:
		return a, tea.Batch(fetchItems(a.client), tick())

	case itemsMsg:
		if msg.err != nil {
			a.statusMsg = "daemon error: " + msg.err.Error()
		} else {
			a.items = msg.items
			// Keep cursor in bounds.
			filtered := a.visibleItems()
			if a.cursor >= len(filtered) && len(filtered) > 0 {
				a.cursor = len(filtered) - 1
			}
		}

	case tea.KeyMsg:
		if a.state == stateForm {
			return a.updateForm(msg)
		}
		return a.updateList(msg)
	}
	return a, nil
}

func (a *App) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	items := a.visibleItems()

	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("q", "ctrl+c"))):
		return a, tea.Quit

	case key.Matches(msg, key.NewBinding(key.WithKeys("1"))):
		a.activeTab = 0
		a.cursor = 0
	case key.Matches(msg, key.NewBinding(key.WithKeys("2"))):
		a.activeTab = 1
		a.cursor = 0
	case key.Matches(msg, key.NewBinding(key.WithKeys("3"))):
		a.activeTab = 2
		a.cursor = 0

	case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
		a.activeTab = (a.activeTab + 1) % len(a.tabs)
		a.cursor = 0
	case key.Matches(msg, key.NewBinding(key.WithKeys("shift+tab"))):
		a.activeTab = (a.activeTab - 1 + len(a.tabs)) % len(a.tabs)
		a.cursor = 0

	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		if a.cursor > 0 {
			a.cursor--
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
		if a.cursor < len(items)-1 {
			a.cursor++
		}

	case key.Matches(msg, key.NewBinding(key.WithKeys("n"))):
		switch tabKinds[a.activeTab] {
		case ipc.KindTimer:
			a.form = newTimerForm()
		case ipc.KindAlarm:
			a.form = newAlarmForm()
		case ipc.KindStopwatch:
			a.form = newStopwatchForm()
		}
		a.state = stateForm

	case key.Matches(msg, key.NewBinding(key.WithKeys("e"))):
		if len(items) > 0 {
			a.form = editForm(items[a.cursor])
			a.state = stateForm
		}

	case key.Matches(msg, key.NewBinding(key.WithKeys("d"))):
		if len(items) > 0 {
			it := items[a.cursor]
			a.client.Delete(it.ID) //nolint
			return a, fetchItems(a.client)
		}

	case key.Matches(msg, key.NewBinding(key.WithKeys("enter", " "))):
		if len(items) > 0 {
			it := items[a.cursor]
			switch it.State {
			case ipc.StateRunning:
				if it.Kind != ipc.KindAlarm {
					a.client.Pause(it.ID) //nolint
				}
			case ipc.StateIdle, ipc.StatePaused:
				a.client.Start(it.ID) //nolint
			}
			return a, fetchItems(a.client)
		}

	case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
		if len(items) > 0 {
			it := items[a.cursor]
			if it.Kind == ipc.KindStopwatch || it.State == ipc.StateDone {
				a.client.Reset(it.ID) //nolint
				return a, fetchItems(a.client)
			}
		}
	}
	return a, nil
}

func (a *App) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.state = stateList
		return a, nil

	case "enter":
		// Only submit when on the last field (or single-field forms).
		if a.form.focused == len(a.form.fields)-1 {
			item, ok := a.form.Submit()
			if !ok {
				// Validation failed; re-render with error.
				return a, nil
			}
			if a.form.mode == formEdit {
				a.client.Edit(item) //nolint
			} else {
				a.client.Create(item) //nolint
			}
			a.state = stateList
			return a, fetchItems(a.client)
		}
		// Move to next field.
	}

	var cmd tea.Cmd
	a.form, cmd = a.form.Update(msg)
	return a, cmd
}

func (a *App) View() string {
	if a.width == 0 {
		return ""
	}

	// Tab bar
	var tabViews []string
	for i, name := range a.tabs {
		label := fmt.Sprintf(" %d: %s ", i+1, name)
		if i == a.activeTab {
			tabViews = append(tabViews, styleActiveTab.Render(label))
		} else {
			tabViews = append(tabViews, styleTab.Render(label))
		}
	}
	tabBar := styleTabBar.Width(a.width).Render(
		lipgloss.JoinHorizontal(lipgloss.Top, tabViews...),
	)

	// Item list
	items := a.visibleItems()
	var listLines []string
	if len(items) == 0 {
		listLines = append(listLines, styleIdle.Render("  No items yet — press n to create one"))
	}
	for i, it := range items {
		prefix := "  "
		if i == a.cursor {
			prefix = styleActive.Render("> ")
		}
		var row string
		switch it.Kind {
		case ipc.KindTimer:
			row = renderTimerItem(it)
		case ipc.KindAlarm:
			row = renderAlarmItem(it)
		case ipc.KindStopwatch:
			row = renderStopwatchItem(it)
		}
		listLines = append(listLines, prefix+row)
	}
	list := strings.Join(listLines, "\n")

	// Help bar
	helpText := a.helpText()
	help := styleHelp.Render(helpText)

	// Status message (errors)
	status := ""
	if a.statusMsg != "" {
		status = "\n" + styleError.Render(a.statusMsg)
	}

	body := tabBar + "\n" + list + status + "\n" + help

	if a.state == stateForm {
		// Center the modal overlay.
		modal := a.form.View(a.width)
		modalLines := strings.Count(modal, "\n") + 1
		bodyLines := strings.Count(body, "\n") + 1
		topPad := (bodyLines - modalLines) / 2
		if topPad < 0 {
			topPad = 0
		}
		leftPad := (a.width - lipgloss.Width(modal)) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		padding := strings.Repeat("\n", topPad) + strings.Repeat(" ", leftPad)
		return body[:0] + padding + strings.ReplaceAll(modal, "\n", "\n"+strings.Repeat(" ", leftPad))
	}

	return body
}

func (a *App) visibleItems() []ipc.Item {
	kind := tabKinds[a.activeTab]
	var out []ipc.Item
	for _, it := range a.items {
		if it.Kind == kind {
			out = append(out, it)
		}
	}
	return out
}

func (a *App) helpText() string {
	base := "n: new  e: edit  d: delete  tab/1-3: switch tab  q: quit"
	switch tabKinds[a.activeTab] {
	case ipc.KindTimer, ipc.KindStopwatch:
		base += "  enter/space: start/pause"
	}
	base += "  r: reset"
	return base
}

// --- Commands ---

func tick() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

func fetchItems(client *ipc.Client) tea.Cmd {
	return func() tea.Msg {
		items, err := client.GetAll()
		return itemsMsg{items: items, err: err}
	}
}

var styleActive = lipgloss.NewStyle().Foreground(colorActive).Bold(true)
