package tcp

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/kienlt/lazy-tests/internal/driver"
)

type Driver struct{}

func New() *Driver {
	return &Driver{}
}

func (d *Driver) Name() string {
	return "tcp"
}

func (d *Driver) Connect(ctx context.Context, target driver.Target, timeout time.Duration) (driver.Session, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctxWithTimeout, "tcp", fmt.Sprintf("%s:%d", target.Host, target.Port))
	if err != nil {
		return nil, err
	}

	return &session{conn: conn}, nil
}

type session struct {
	conn net.Conn
}

func (s *session) Close() error {
	if s.conn == nil {
		return nil
	}
	return s.conn.Close()
}
