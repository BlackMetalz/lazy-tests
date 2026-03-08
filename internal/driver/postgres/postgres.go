package postgres

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/kienlt/lazy-tests/internal/driver"
)

type Driver struct{}

func New() *Driver {
	return &Driver{}
}

func (d *Driver) Name() string {
	return "postgres"
}

func (d *Driver) Connect(ctx context.Context, target driver.Target, timeout time.Duration) (driver.Session, error) {
	user := target.Auth.Username
	if user == "" {
		user = "postgres"
	}

	database := target.Auth.Database
	if database == "" {
		database = "postgres"
	}

	dsnURL := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, target.Auth.Password),
		Host:   net.JoinHostPort(target.Host, strconv.Itoa(target.Port)),
		Path:   database,
	}
	q := dsnURL.Query()
	timeoutSeconds := int(timeout.Seconds())
	if timeoutSeconds < 1 {
		timeoutSeconds = 1
	}
	q.Set("connect_timeout", strconv.Itoa(timeoutSeconds))
	dsnURL.RawQuery = q.Encode()

	cfg, err := pgx.ParseConfig(dsnURL.String())
	if err != nil {
		return nil, err
	}

	connectCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	conn, err := pgx.ConnectConfig(connectCtx, cfg)
	if err != nil {
		return nil, fmt.Errorf("postgres connect: %w", err)
	}

	return &session{conn: conn}, nil
}

type session struct {
	conn *pgx.Conn
}

func (s *session) Close() error {
	if s.conn == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return s.conn.Close(ctx)
}
