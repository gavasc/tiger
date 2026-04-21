package waybar

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gaio/tiger/internal/ipc"
)

type output struct {
	Text    string `json:"text"`
	Tooltip string `json:"tooltip,omitempty"`
	Class   string `json:"class"`
}

// Print writes a single waybar JSON line to stdout.
// Empty output makes waybar hide the module entirely.
func Print(client *ipc.Client) {
	items, err := client.GetAll()
	if err != nil {
		// Daemon not running — show a warning icon.
		emit(output{Text: "⚠", Class: "error"})
		return
	}

	var active []ipc.Item
	for _, it := range items {
		if it.State == ipc.StateRunning || it.State == ipc.StatePaused {
			active = append(active, it)
		}
	}

	if len(active) == 0 {
		// Empty text = waybar hides the module.
		fmt.Println("")
		return
	}

	var lines []string
	for _, it := range active {
		var stateIcon string
		if it.State == ipc.StatePaused {
			stateIcon = "⏸ "
		}
		switch it.Kind {
		case ipc.KindTimer:
			lines = append(lines, fmt.Sprintf("%s%s  %s left", stateIcon, it.Name, formatDuration(it.Remaining())))
		case ipc.KindAlarm:
			lines = append(lines, fmt.Sprintf("⏰ %s  at %s", it.Name, it.TargetAt.Format("15:04")))
		case ipc.KindStopwatch:
			lines = append(lines, fmt.Sprintf("%s%s  %s", stateIcon, it.Name, formatDuration(it.Elapsed)))
		}
	}

	emit(output{
		Text:    fmt.Sprintf("󱎫  %d", len(active)), // nerd font clock icon
		Tooltip: strings.Join(lines, "\n"),
		Class:   "active",
	})
}

func emit(o output) {
	data, _ := json.Marshal(o)
	fmt.Println(string(data))
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
