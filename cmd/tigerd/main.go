package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/gaio/tiger/internal/daemon"
)

func socketPath() string {
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "tiger")
		os.MkdirAll(dir, 0o700)
	}
	return filepath.Join(dir, "tiger.sock")
}

func main() {
	d, err := daemon.New()
	if err != nil {
		log.Fatalf("tigerd: init: %v", err)
	}

	sock := socketPath()
	log.Printf("tigerd: listening on %s", sock)

	if err := d.Run(sock); err != nil {
		log.Fatalf("tigerd: %v", err)
	}
}
