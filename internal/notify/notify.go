package notify

import (
	"os/exec"
)

const (
	alarmSound    = "/usr/share/sounds/freedesktop/stereo/alarm-clock-elapsed.oga"
	fallbackSound = "/usr/share/sounds/freedesktop/stereo/complete.oga"
)

// Fire sends a desktop notification and plays 4 seconds of alarm sound.
func Fire(name string) {
	go func() {
		exec.Command("notify-send", "-u", "critical", "-a", "tiger", "-i", "alarm", "tiger", name).Run()
	}()
	go func() {
		if err := exec.Command("ffplay",
			"-nodisp", "-autoexit", "-loglevel", "quiet",
			"-t", "4",
			"-af", "volume=1.5",
			alarmSound,
		).Run(); err != nil {
			exec.Command("paplay", "--volume=65536", fallbackSound).Run()
		}
	}()
}
