package redis

import (
	"context"
	"net"
	"strconv"
	"time"

	"github.com/kienlt/lazy-tests/internal/driver"
	redisv9 "github.com/redis/go-redis/v9"
)

type Driver struct{}

func New() *Driver {
	return &Driver{}
}

func (d *Driver) Name() string {
	return "redis"
}

func (d *Driver) Connect(ctx context.Context, target driver.Target, timeout time.Duration) (driver.Session, error) {
	client := redisv9.NewClient(&redisv9.Options{
		Addr:         net.JoinHostPort(target.Host, strconv.Itoa(target.Port)),
		Password:     target.Auth.Password,
		DB:           target.Auth.RedisDB,
		DialTimeout:  timeout,
		ReadTimeout:  timeout,
		WriteTimeout: timeout,
	})

	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}

	return &session{client: client}, nil
}

type session struct {
	client *redisv9.Client
}

func (s *session) Close() error {
	if s.client == nil {
		return nil
	}
	return s.client.Close()
}
