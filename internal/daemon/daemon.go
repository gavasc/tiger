package daemon

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/gavasc/tiger/internal/ipc"
	"github.com/gavasc/tiger/internal/notify"
	"github.com/google/uuid"
)

// Daemon holds all item state and drives the tick loop.
type Daemon struct {
	mu    sync.Mutex
	items []ipc.Item
}

func New() (*Daemon, error) {
	items, err := loadState()
	if err != nil {
		log.Printf("daemon: failed to load state: %v (starting fresh)", err)
		items = nil
	}
	// Reset any running items to paused on restart so they don't silently advance.
	for i := range items {
		if items[i].State == ipc.StateRunning {
			items[i].State = ipc.StatePaused
		}
	}
	return &Daemon{items: items}, nil
}

// Run starts the IPC server and tick loop.
func (d *Daemon) Run(socketPath string) error {
	// Remove stale socket if present.
	os.Remove(socketPath)

	go d.tickLoop()

	return ipc.Serve(socketPath, d.handle)
}

func (d *Daemon) tickLoop() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		d.tick()
	}
}

func (d *Daemon) tick() {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	changed := false

	for i := range d.items {
		it := &d.items[i]
		if it.State != ipc.StateRunning {
			continue
		}

		switch it.Kind {
		case ipc.KindTimer:
			it.Elapsed += 500 * time.Millisecond
			if it.Elapsed >= it.Duration {
				it.Elapsed = it.Duration
				it.State = ipc.StateDone
				notify.Fire(it.Name + " timer finished")
			}
			changed = true

		case ipc.KindAlarm:
			if now.After(it.TargetAt) {
				it.State = ipc.StateDone
				notify.Fire(it.Name + " alarm!")
				changed = true
			}

		case ipc.KindStopwatch:
			it.Elapsed += 500 * time.Millisecond
			changed = true
		}
	}

	if changed {
		if err := saveState(d.items); err != nil {
			log.Printf("daemon: save state: %v", err)
		}
	}
}

func (d *Daemon) handle(cmd ipc.Command) ipc.Response {
	d.mu.Lock()
	defer d.mu.Unlock()

	switch cmd.Op {
	case ipc.OpGetAll:
		items := make([]ipc.Item, len(d.items))
		copy(items, d.items)
		return ipc.Response{Items: items}

	case ipc.OpCreate:
		item := cmd.Item
		item.ID = uuid.NewString()
		item.CreatedAt = time.Now()
		item.State = ipc.StateIdle
		if item.Kind == ipc.KindAlarm {
			item.State = ipc.StateRunning // alarms are always "armed"
		}
		d.items = append(d.items, item)
		d.save()
		return ipc.Response{Items: d.copyItems()}

	case ipc.OpEdit:
		for i := range d.items {
			if d.items[i].ID == cmd.Item.ID {
				// Preserve runtime state, update editable fields only.
				d.items[i].Name = cmd.Item.Name
				d.items[i].Duration = cmd.Item.Duration
				d.items[i].TargetAt = cmd.Item.TargetAt
				break
			}
		}
		d.save()
		return ipc.Response{Items: d.copyItems()}

	case ipc.OpDelete:
		for i, it := range d.items {
			if it.ID == cmd.ID {
				d.items = append(d.items[:i], d.items[i+1:]...)
				break
			}
		}
		d.save()
		return ipc.Response{Items: d.copyItems()}

	case ipc.OpStart:
		for i := range d.items {
			if d.items[i].ID == cmd.ID {
				d.items[i].State = ipc.StateRunning
				break
			}
		}
		d.save()
		return ipc.Response{Items: d.copyItems()}

	case ipc.OpPause:
		for i := range d.items {
			if d.items[i].ID == cmd.ID {
				d.items[i].State = ipc.StatePaused
				break
			}
		}
		d.save()
		return ipc.Response{Items: d.copyItems()}

	case ipc.OpReset:
		for i := range d.items {
			if d.items[i].ID == cmd.ID {
				d.items[i].Elapsed = 0
				if d.items[i].Kind == ipc.KindAlarm {
					// Re-arm for the next occurrence of the same HH:MM.
					prev := d.items[i].TargetAt
					now := time.Now()
					next := time.Date(now.Year(), now.Month(), now.Day(),
						prev.Hour(), prev.Minute(), 0, 0, now.Location())
					if !next.After(now) {
						next = next.Add(24 * time.Hour)
					}
					d.items[i].TargetAt = next
					d.items[i].State = ipc.StateRunning
				} else {
					d.items[i].State = ipc.StateIdle
				}
				break
			}
		}
		d.save()
		return ipc.Response{Items: d.copyItems()}
	}

	return ipc.Response{Error: "unknown op: " + string(cmd.Op)}
}

func (d *Daemon) copyItems() []ipc.Item {
	items := make([]ipc.Item, len(d.items))
	copy(items, d.items)
	return items
}

func (d *Daemon) save() {
	if err := saveState(d.items); err != nil {
		log.Printf("daemon: save state: %v", err)
	}
}
