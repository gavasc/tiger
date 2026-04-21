package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gaio/tiger/internal/ipc"
	"github.com/gaio/tiger/internal/tui"
	"github.com/gaio/tiger/internal/waybar"
)

func socketPath() string {
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "tiger")
		os.MkdirAll(dir, 0o700)
	}
	return filepath.Join(dir, "tiger.sock")
}

func isDaemonRunning(sock string) bool {
	conn, err := net.DialTimeout("unix", sock, 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// findTigerd returns the path to tigerd, checking the same dir as the
// current binary first, then $PATH.
func findTigerd() (string, error) {
	self, err := os.Executable()
	if err == nil {
		candidate := filepath.Join(filepath.Dir(self), "tigerd")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return exec.LookPath("tigerd")
}

func spawnDaemon() error {
	path, err := findTigerd()
	if err != nil {
		return fmt.Errorf("tigerd not found in $PATH or beside tiger binary: %w", err)
	}
	cmd := exec.Command(path)
	cmd.Stdout = nil
	cmd.Stderr = nil
	// New session so tigerd is not killed when the terminal or tiger exits.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start tigerd: %w", err)
	}
	cmd.Process.Release()
	return nil
}

func ensureDaemon(sock string) error {
	if isDaemonRunning(sock) {
		return nil
	}

	if err := spawnDaemon(); err != nil {
		return err
	}

	// Poll until socket is ready (up to 3 seconds).
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
		if isDaemonRunning(sock) {
			return nil
		}
	}
	return fmt.Errorf("tigerd did not start in time")
}

func installSystemdService() {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return
	}

	tigerdPath, err := findTigerd()
	if err != nil {
		return
	}

	home, _ := os.UserHomeDir()
	marker := filepath.Join(home, ".local", "share", "tiger", ".installed")
	if _, err := os.Stat(marker); err == nil {
		return
	}

	serviceDir := filepath.Join(home, ".config", "systemd", "user")
	os.MkdirAll(serviceDir, 0o755)

	systemService := "/usr/lib/systemd/user/tigerd.service"
	if _, err := os.Stat(systemService); os.IsNotExist(err) {
		svcContent := fmt.Sprintf(`[Unit]
Description=Tiger alarm/timer daemon

[Service]
ExecStart=%s
Restart=on-failure

[Install]
WantedBy=default.target
`, tigerdPath)
		servicePath := filepath.Join(serviceDir, "tigerd.service")
		if err := os.WriteFile(servicePath, []byte(svcContent), 0o644); err != nil {
			return
		}
	}

	exec.Command("systemctl", "--user", "enable", "--now", "tigerd").Run()

	os.MkdirAll(filepath.Dir(marker), 0o755)
	os.WriteFile(marker, []byte(""), 0o644)

	setupWaybar()
}

// waybarSnippet is the module definition the user needs to add to their waybar config.
const waybarSnippet = `"custom/tiger": {
    "exec": "tiger waybar",
    "interval": 2,
    "return-type": "json",
    "on-click": "tiger",
    "tooltip": true,
    "format": "{}"
}`

// waybarCSS is optional styling for the module.
const waybarCSS = `
#custom-tiger {
    color: #29b6f6;
    margin: 0 4px;
}
#custom-tiger.active {
    color: #00e676;
}
#custom-tiger.error {
    color: #ff5252;
}
`

func setupWaybar() {
	// Only if waybar is installed.
	if _, err := exec.LookPath("waybar"); err != nil {
		return
	}

	home, _ := os.UserHomeDir()
	waybarDir := filepath.Join(home, ".config", "waybar")
	os.MkdirAll(waybarDir, 0o755)

	snippetPath := filepath.Join(waybarDir, "tiger-module.jsonc")
	cssPath := filepath.Join(waybarDir, "tiger.css")

	// Write the JSON snippet (safe to overwrite — it's reference material).
	os.WriteFile(snippetPath, []byte(waybarSnippet+"\n"), 0o644)
	os.WriteFile(cssPath, []byte(waybarCSS), 0o644)

	// Print one-time setup instructions to stderr so they appear above the TUI.
	fmt.Fprintf(os.Stderr, "\n"+
		"tiger: waybar module ready!\n"+
		"  1. Add \"custom/tiger\" to your modules list in ~/.config/waybar/config.jsonc\n"+
		"  2. Paste the module config from ~/.config/waybar/tiger-module.jsonc\n"+
		"  3. Optionally @import \"tiger.css\" in your waybar style.css\n"+
		"  4. Reload waybar (killall -SIGUSR2 waybar)\n\n")
}

func main() {
	// Handle subcommand: tiger waybar
	// Outputs a single JSON line for waybar's custom module and exits.
	if len(os.Args) == 2 && os.Args[1] == "waybar" {
		sock := socketPath()
		client := ipc.NewClient(sock)
		waybar.Print(client)
		return
	}

	sock := socketPath()

	if err := ensureDaemon(sock); err != nil {
		fmt.Fprintf(os.Stderr, "tiger: %v\n", err)
		os.Exit(1)
	}

	// Best-effort systemd + waybar setup on first run (non-fatal).
	go installSystemdService()

	client := ipc.NewClient(sock)
	app := tui.NewApp(client)

	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tiger: %v\n", err)
		os.Exit(1)
	}
}
