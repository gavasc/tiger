package tui

import (
	"fmt"
	"time"

	"github.com/gavasc/tiger/internal/ipc"
)

func renderTimerItem(it ipc.Item) string {
	var stateStr string
	var stateStyle = styleIdle
	switch it.State {
	case ipc.StateRunning:
		stateStr = "▶"
		stateStyle = styleRunning
	case ipc.StatePaused:
		stateStr = "⏸"
		stateStyle = stylePaused
	case ipc.StateDone:
		stateStr = "✓"
		stateStyle = styleDone
	default:
		stateStr = "·"
	}

	rem := it.Remaining()
	timeStr := formatDuration(rem)
	if it.State == ipc.StateDone {
		timeStr = "done"
	}

	return fmt.Sprintf("%s  %-20s  %s / %s",
		stateStyle.Render(stateStr),
		it.Name,
		stateStyle.Render(timeStr),
		formatDuration(it.Duration),
	)
}

func renderAlarmItem(it ipc.Item) string {
	var stateStr string
	var stateStyle = styleIdle
	switch it.State {
	case ipc.StateRunning:
		stateStr = "⏰"
		stateStyle = styleRunning
	case ipc.StateDone:
		stateStr = "✓"
		stateStyle = styleDone
	default:
		stateStr = "·"
	}

	timeStr := it.TargetAt.Format("15:04")
	if it.State == ipc.StateDone {
		timeStr = "fired"
	}

	return fmt.Sprintf("%s  %-20s  %s",
		stateStyle.Render(stateStr),
		it.Name,
		stateStyle.Render(timeStr),
	)
}

func renderStopwatchItem(it ipc.Item) string {
	var stateStr string
	var stateStyle = styleIdle
	switch it.State {
	case ipc.StateRunning:
		stateStr = "▶"
		stateStyle = styleRunning
	case ipc.StatePaused:
		stateStr = "⏸"
		stateStyle = stylePaused
	default:
		stateStr = "·"
	}

	return fmt.Sprintf("%s  %-20s  %s",
		stateStyle.Render(stateStr),
		it.Name,
		stateStyle.Render(formatDuration(it.Elapsed)),
	)
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}
