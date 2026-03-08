package driver

import (
	"context"
	"time"

	"github.com/kienlt/lazy-tests/internal/scenario"
)

type Target struct {
	Protocol string
	Host     string
	Port     int
	Auth     scenario.Auth
}

type Session interface {
	Close() error
}

type Driver interface {
	Name() string
	Connect(ctx context.Context, target Target, timeout time.Duration) (Session, error)
}
