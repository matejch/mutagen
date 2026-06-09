package engine

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// watchMemory polls the RSS of a process and its children. If total RSS
// exceeds limitBytes, it kills the process. Runs until the done channel
// is closed or the process is killed.
func watchMemory(pid int, limitBytes uint64, done <-chan struct{}, onKill func()) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			rss := getProcessTreeRSS(pid)
			if rss > 0 && rss > limitBytes {
				if onKill != nil {
					onKill()
				}
				// Kill the process group
				exec.Command("kill", "-9", strconv.Itoa(pid)).Run() //nolint:errcheck
				return
			}
		}
	}
}

// getProcessTreeRSS returns the total RSS in bytes of a process and all its
// descendants. Uses `ps` which works on both macOS and Linux.
func getProcessTreeRSS(pid int) uint64 {
	// ps -o rss= -p PID gives RSS in KB for that process
	// For child processes, use ps -o rss= --ppid PID on Linux
	// On macOS, use ps -o rss=,ppid= -ax and filter manually
	//
	// Simplest cross-platform: just check the main process.
	// go test spawns the test binary as a child, so we check the whole group.
	out, err := exec.Command("ps", "-o", "rss=", "-g", strconv.Itoa(pid)).Output()
	if err != nil {
		// Fallback: just check the single process
		out, err = exec.Command("ps", "-o", "rss=", "-p", strconv.Itoa(pid)).Output()
		if err != nil {
			return 0
		}
	}

	var total uint64
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		kb, err := strconv.ParseUint(line, 10, 64)
		if err != nil {
			continue
		}
		total += kb * 1024
	}
	return total
}

// parseMemLimit parses a human-readable memory limit like "512MiB", "1GiB", "256MB".
func parseMemLimit(s string) (uint64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}

	multipliers := map[string]uint64{
		"B":   1,
		"KB":  1000,
		"KiB": 1024,
		"MB":  1000 * 1000,
		"MiB": 1024 * 1024,
		"GB":  1000 * 1000 * 1000,
		"GiB": 1024 * 1024 * 1024,
	}

	for suffix, mult := range multipliers {
		if strings.HasSuffix(s, suffix) {
			numStr := strings.TrimSuffix(s, suffix)
			num, err := strconv.ParseFloat(strings.TrimSpace(numStr), 64)
			if err != nil {
				return 0, fmt.Errorf("invalid memory limit %q", s)
			}
			return uint64(num * float64(mult)), nil
		}
	}

	// Try plain number as bytes
	num, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid memory limit %q (use e.g. 512MiB, 1GiB)", s)
	}
	return num, nil
}
