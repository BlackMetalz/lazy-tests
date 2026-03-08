package engine

import (
	"context"
	"errors"
	"fmt"
	"net"
	"syscall"
	"testing"
)

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "timeout by context", err: context.DeadlineExceeded, want: "timeout"},
		{name: "dns error", err: &net.DNSError{Err: "no such host"}, want: "dns"},
		{name: "refused wrapped", err: fmt.Errorf("wrapped: %w", syscall.ECONNREFUSED), want: "refused"},
		{name: "reset wrapped", err: fmt.Errorf("wrapped: %w", syscall.ECONNRESET), want: "reset"},
		{name: "other", err: errors.New("boom"), want: "other"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyError(tt.err)
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
