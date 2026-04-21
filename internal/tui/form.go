package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gaio/tiger/internal/ipc"
)

type formMode int

const (
	formNew  formMode = iota
	formEdit
)

// Form is an overlay modal for creating/editing an item.
type Form struct {
	kind    ipc.ItemKind
	mode    formMode
	editID  string
	fields  []textinput.Model
	focused int
	err     string
}

func newTimerForm() Form {
	name := textinput.New()
	name.Placeholder = "e.g. Tea"
	name.Focus()

	dur := textinput.New()
	dur.Placeholder = "e.g. 5m30s or 1h"

	return Form{kind: ipc.KindTimer, fields: []textinput.Model{name, dur}}
}

func newAlarmForm() Form {
	name := textinput.New()
	name.Placeholder = "e.g. Meeting"
	name.Focus()

	t := textinput.New()
	t.Placeholder = "HH:MM (24h)"

	return Form{kind: ipc.KindAlarm, fields: []textinput.Model{name, t}}
}

func newStopwatchForm() Form {
	name := textinput.New()
	name.Placeholder = "e.g. Sprint"
	name.Focus()

	return Form{kind: ipc.KindStopwatch, fields: []textinput.Model{name}}
}

func editForm(it ipc.Item) Form {
	var f Form
	switch it.Kind {
	case ipc.KindTimer:
		f = newTimerForm()
		f.fields[0].SetValue(it.Name)
		f.fields[1].SetValue(it.Duration.String())
	case ipc.KindAlarm:
		f = newAlarmForm()
		f.fields[0].SetValue(it.Name)
		f.fields[1].SetValue(it.TargetAt.Format("15:04"))
	case ipc.KindStopwatch:
		f = newStopwatchForm()
		f.fields[0].SetValue(it.Name)
	}
	f.mode = formEdit
	f.editID = it.ID
	return f
}

func (f Form) fieldLabels() []string {
	switch f.kind {
	case ipc.KindTimer:
		return []string{"Name", "Duration"}
	case ipc.KindAlarm:
		return []string{"Name", "Time (HH:MM)"}
	case ipc.KindStopwatch:
		return []string{"Name"}
	}
	return nil
}

func (f Form) Update(msg tea.Msg) (Form, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			f.err = ""
			f.fields[f.focused].Blur()
			f.focused = (f.focused + 1) % len(f.fields)
			f.fields[f.focused].Focus()
		case "shift+tab", "up":
			f.err = ""
			f.fields[f.focused].Blur()
			f.focused = (f.focused - 1 + len(f.fields)) % len(f.fields)
			f.fields[f.focused].Focus()
		}
	}

	var cmd tea.Cmd
	f.fields[f.focused], cmd = f.fields[f.focused].Update(msg)
	return f, cmd
}

func (f Form) View(width int) string {
	title := "New "
	if f.mode == formEdit {
		title = "Edit "
	}
	switch f.kind {
	case ipc.KindTimer:
		title += "Timer"
	case ipc.KindAlarm:
		title += "Alarm"
	case ipc.KindStopwatch:
		title += "Stopwatch"
	}

	labels := f.fieldLabels()
	var rows []string
	rows = append(rows, lipgloss.NewStyle().Bold(true).Foreground(colorActive).Render(title))
	rows = append(rows, "")
	for i, field := range f.fields {
		label := styleLabel.Render(labels[i])
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, label, field.View()))
	}
	if f.err != "" {
		rows = append(rows, "", styleError.Render(f.err))
	}
	rows = append(rows, "", styleHelp.Render("tab: next field • enter: confirm • esc: cancel"))

	content := strings.Join(rows, "\n")

	maxW := width - 8
	if maxW < 40 {
		maxW = 40
	}
	return styleModal.Width(maxW).Render(content)
}

// Submit validates and returns the Item to create/edit, or sets f.err and returns false.
func (f *Form) Submit() (ipc.Item, bool) {
	name := strings.TrimSpace(f.fields[0].Value())
	if name == "" {
		f.err = "name cannot be empty"
		return ipc.Item{}, false
	}

	item := ipc.Item{
		ID:   f.editID,
		Kind: f.kind,
		Name: name,
	}

	switch f.kind {
	case ipc.KindTimer:
		raw := strings.TrimSpace(f.fields[1].Value())
		d, err := time.ParseDuration(raw)
		if err != nil || d <= 0 {
			f.err = fmt.Sprintf("invalid duration %q — use e.g. 5m or 1h30m", raw)
			return ipc.Item{}, false
		}
		item.Duration = d

	case ipc.KindAlarm:
		raw := strings.TrimSpace(f.fields[1].Value())
		t, err := time.Parse("15:04", raw)
		if err != nil {
			f.err = fmt.Sprintf("invalid time %q — use HH:MM (24h)", raw)
			return ipc.Item{}, false
		}
		now := time.Now()
		target := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())
		if target.Before(now) {
			target = target.Add(24 * time.Hour)
		}
		item.TargetAt = target
	}

	return item, true
}
