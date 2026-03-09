package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	listenAddr := flag.String("listen", ":18080", "TCP listen address")
	readTimeout := flag.Duration("read-timeout", 2*time.Second, "read timeout per connection")
	hold := flag.Duration("hold", 0, "how long to keep connection open before closing")
	reply := flag.String("reply", "ok\n", "reply payload")
	writeDelay := flag.Duration("write-delay", 0, "delay before writing reply")
	leakCloseWait := flag.Bool("leak-close-wait", false, "intentionally keep sockets in CLOSE_WAIT after client close")
	leakLimit := flag.Int("leak-limit", 5000, "maximum leaked CLOSE_WAIT sockets to keep")
	statsEvery := flag.Duration("stats-every", 5*time.Second, "stats log interval")
	logEvery := flag.Int("log-every", 100, "log every N accepts/leaks; set 0 to disable per-connection progress logs")
	flag.Parse()

	ln, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatalf("listen error: %v", err)
	}
	defer ln.Close()

	log.Printf("high-conntrack server listening on %s", *listenAddr)
	if *leakCloseWait {
		log.Printf("CLOSE_WAIT leak mode enabled (limit=%d)", *leakLimit)
	}

	var totalAccepted atomic.Int64
	var active atomic.Int64

	var leakedMu sync.Mutex
	leaked := make([]net.Conn, 0, max(0, *leakLimit))

	go func() {
		ticker := time.NewTicker(*statsEvery)
		defer ticker.Stop()
		for range ticker.C {
			leakedMu.Lock()
			leakedCount := len(leaked)
			leakedMu.Unlock()
			log.Printf("stats accepted=%d active=%d leaked_close_wait=%d", totalAccepted.Load(), active.Load(), leakedCount)
		}
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			continue
		}

		accepted := totalAccepted.Add(1)
		active.Add(1)
		if *logEvery > 0 && (accepted == 1 || accepted%int64(*logEvery) == 0) {
			log.Printf("accept #%d remote=%s local=%s", accepted, conn.RemoteAddr(), conn.LocalAddr())
		}

		go func(c net.Conn) {
			defer active.Add(-1)

			if *leakCloseWait {
				handleLeakCloseWait(c, *readTimeout, *leakLimit, &leakedMu, &leaked, *logEvery)
				return
			}

			defer c.Close()
			_ = c.SetReadDeadline(time.Now().Add(*readTimeout))
			buf := make([]byte, 1024)
			_, err := c.Read(buf)
			if err != nil && !errors.Is(err, io.EOF) {
				return
			}

			if *hold > 0 {
				time.Sleep(*hold)
			}
			if *writeDelay > 0 {
				time.Sleep(*writeDelay)
			}
			if *reply != "" {
				_, _ = io.WriteString(c, *reply)
			}
		}(conn)
	}
}

func handleLeakCloseWait(c net.Conn, readTimeout time.Duration, leakLimit int, leakedMu *sync.Mutex, leaked *[]net.Conn, logEvery int) {
	_ = c.SetReadDeadline(time.Now().Add(readTimeout))
	buf := make([]byte, 1024)
	for {
		_, err := c.Read(buf)
		if err == nil {
			continue
		}

		if errors.Is(err, io.EOF) {
			leakedMu.Lock()
			if len(*leaked) < leakLimit {
				*leaked = append(*leaked, c)
				leakedCount := len(*leaked)
				leakedMu.Unlock()
				if logEvery > 0 && (leakedCount == 1 || leakedCount%logEvery == 0) {
					log.Printf("leaked CLOSE_WAIT #%d remote=%s local=%s", leakedCount, c.RemoteAddr(), c.LocalAddr())
				}
				return
			}
			leakedMu.Unlock()
		}

		_ = c.Close()
		return
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func init() {
	flag.Usage = func() {
		fmt.Println("High-conntrack / CLOSE_WAIT lab server")
		flag.PrintDefaults()
	}
}
