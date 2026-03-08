package engine

import (
	"context"
	"errors"
	"net"
	"strings"
	"syscall"
)

func ClassifyError(err error) string {
	if err == nil {
		return ""
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "timeout"
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return "dns"
	}

	if errors.Is(err, syscall.ECONNREFUSED) {
		return "refused"
	}

	if errors.Is(err, syscall.ECONNRESET) {
		return "reset"
	}

	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "connection refused") {
		return "refused"
	}
	if strings.Contains(msg, "connection reset") {
		return "reset"
	}
	if strings.Contains(msg, "i/o timeout") || strings.Contains(msg, "timeout") {
		return "timeout"
	}
	if strings.Contains(msg, "no such host") {
		return "dns"
	}

	return "other"
}
