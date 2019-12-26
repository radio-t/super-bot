package events

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"time"

	log "github.com/go-pkgz/lgr"
)

// Rtjc for rtjc commands. Publishes whatever it got from socket
// compatible with the legacy rtjc bot, used to push news events from news.radio-t.com
type Rtjc struct {
	Port      int
	Submitter Submitter
}

type Submitter interface {
	Submit(ctx context.Context, msg string) error
}

// Listen on Port accept and forward to gitter
func (l Rtjc) Listen(ctx context.Context) {
	log.Printf("[INFO] rtjc listener on port %d", l.Port)
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", l.Port))
	if err != nil {
		log.Fatalf("[ERROR] can't listen on %d, %v", l.Port, err)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("[WARN] can't accept, %v", err)
			time.Sleep(time.Second * 1)
			continue
		}
		if message, err := bufio.NewReader(conn).ReadString('\n'); err == nil {
			if e := l.Submitter.Submit(ctx, message); e != nil {
				log.Printf("[WARN] can't send message, %v", err)
			}
		}
		_ = conn.Close()
	}
}
