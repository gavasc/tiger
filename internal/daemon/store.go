package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/gaio/tiger/internal/ipc"
)

func stateFile() string {
	dir, _ := os.UserHomeDir()
	return filepath.Join(dir, ".local", "share", "tiger", "state.json")
}

func loadState() ([]ipc.Item, error) {
	path := stateFile()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var items []ipc.Item
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func saveState(items []ipc.Item) error {
	path := stateFile()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
