package ipc

import "time"

type ItemKind string

const (
	KindTimer     ItemKind = "timer"
	KindAlarm     ItemKind = "alarm"
	KindStopwatch ItemKind = "stopwatch"
)

type ItemState string

const (
	StateIdle    ItemState = "idle"
	StateRunning ItemState = "running"
	StatePaused  ItemState = "paused"
	StateDone    ItemState = "done"
)

type Item struct {
	ID        string        `json:"id"`
	Kind      ItemKind      `json:"kind"`
	Name      string        `json:"name"`
	State     ItemState     `json:"state"`
	TargetAt  time.Time     `json:"target_at,omitempty"`  // alarm: absolute fire time
	Duration  time.Duration `json:"duration,omitempty"`   // timer: total countdown
	Elapsed   time.Duration `json:"elapsed,omitempty"`    // timer: time elapsed; stopwatch: time elapsed
	CreatedAt time.Time     `json:"created_at"`
}

// Remaining returns time left for a running timer.
func (it Item) Remaining() time.Duration {
	if it.Kind != KindTimer {
		return 0
	}
	r := it.Duration - it.Elapsed
	if r < 0 {
		return 0
	}
	return r
}

type Op string

const (
	OpCreate  Op = "create"
	OpEdit    Op = "edit"
	OpDelete  Op = "delete"
	OpStart   Op = "start"
	OpPause   Op = "pause"
	OpReset   Op = "reset"
	OpGetAll  Op = "get_all"
)

type Command struct {
	Op   Op   `json:"op"`
	Item Item `json:"item,omitempty"`
	ID   string `json:"id,omitempty"`
}

type Response struct {
	Items []Item `json:"items"`
	Error string `json:"error,omitempty"`
}
