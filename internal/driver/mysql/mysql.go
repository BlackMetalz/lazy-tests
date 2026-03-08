package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strconv"
	"time"

	mysqlsdk "github.com/go-sql-driver/mysql"
	"github.com/kienlt/lazy-tests/internal/driver"
)

type Driver struct{}

func New() *Driver {
	return &Driver{}
}

func (d *Driver) Name() string {
	return "mysql"
}

func (d *Driver) Connect(ctx context.Context, target driver.Target, timeout time.Duration) (driver.Session, error) {
	cfg := mysqlsdk.NewConfig()
	cfg.Net = "tcp"
	cfg.Addr = net.JoinHostPort(target.Host, strconv.Itoa(target.Port))
	cfg.User = target.Auth.Username
	cfg.Passwd = target.Auth.Password
	cfg.DBName = target.Auth.Database
	cfg.Timeout = timeout
	cfg.ReadTimeout = timeout
	cfg.WriteTimeout = timeout

	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, err
	}

	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("mysql ping: %w", err)
	}

	return &session{db: db}, nil
}

type session struct {
	db *sql.DB
}

func (s *session) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}
