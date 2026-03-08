package engine

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func probeSocketStates(ctx context.Context, host string, port int) SocketStates {
	probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(probeCtx, "netstat", "-an")
	out, err := cmd.Output()
	if err != nil {
		return SocketStates{Available: false, Message: fmt.Sprintf("netstat unavailable: %v", err)}
	}

	var established int
	var timeWait int
	portToken := strconv.Itoa(port)
	host = strings.ToLower(host)

	for _, line := range bytes.Split(out, []byte("\n")) {
		lower := strings.ToLower(string(line))
		if lower == "" {
			continue
		}

		if !lineMatchesTarget(lower, host, portToken) {
			continue
		}

		if strings.Contains(lower, "established") {
			established++
		}
		if strings.Contains(lower, "time_wait") || strings.Contains(lower, "timewait") {
			timeWait++
		}
	}

	return SocketStates{
		Available:   true,
		Established: established,
		TimeWait:    timeWait,
	}
}

func lineMatchesTarget(line, host, port string) bool {
	if strings.Contains(line, host+":"+port) || strings.Contains(line, host+"."+port) {
		return true
	}

	// Best effort fallback for wildcard/local interface formats.
	if strings.Contains(line, ":"+port+" ") || strings.Contains(line, "."+port+" ") {
		return true
	}

	return false
}
