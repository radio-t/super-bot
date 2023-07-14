package events

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/go-pkgz/syncs"
	"golang.org/x/time/rate"
)

//go:generate moq --out mocks/submitter.go --pkg mocks --skip-ensure . submitter:Submitter
//go:generate moq --out mocks/summarizer.go --pkg mocks --skip-ensure . summarizer:Summarizer

// pinned defines translation map for messages pinned by bot
var pinned = map[string]string{
	"⚠️ Официальный кат! - https://stream.radio-t.com/": "⚠️ Вещание подкаста началось - https://stream.radio-t.com/",
}

// Rtjc is a listener for incoming rtjc commands. Publishes whatever it got from the socket
// compatible with the legacy rtjc bot. Primarily use case is to push news events from news.radio-t.com
type Rtjc struct {
	Port       int
	Submitter  submitter
	Summarizer summarizer

	Swg             *syncs.SizedGroup
	SubmitRateLimit rate.Limit
	SubmitRateBurst int
	EnableSummary   bool
}

// submitter defines interface to submit (usually asynchronously) to the chat
type submitter interface {
	Submit(ctx context.Context, text string, pin bool) error
	SubmitHTML(ctx context.Context, text string, pin bool) error
}

type summarizer interface {
	GetSummariesByMessage(remarkLink string) (messages []string, err error)
}

// Listen on Port accept and forward to telegram
func (l Rtjc) Listen(ctx context.Context) {
	log.Printf("[INFO] rtjc listener on port %d", l.Port)
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", l.Port))
	if err != nil {
		log.Fatalf("[ERROR] can't listen on %d, %v", l.Port, err)
	}

	for {
		conn, e := ln.Accept()
		if e != nil {
			log.Printf("[WARN] can't accept, %v", e)
			time.Sleep(time.Second * 1)
			continue
		}
		l.processMessage(ctx, conn)
		_ = conn.Close()
	}
}

func (l Rtjc) processMessage(ctx context.Context, conn io.Reader) {
	if message, rerr := bufio.NewReader(conn).ReadString('\n'); rerr == nil {
		pin, msg := l.isPinned(message)
		if serr := l.Submitter.Submit(ctx, msg, pin); serr != nil {
			log.Printf("[WARN] can't send message, %v", serr)
		}

		l.Swg.Go(func(ctx context.Context) {
			l.sendSummary(ctx, msg)
		})
	} else {
		log.Printf("[WARN] can't read message, %v", rerr)
	}
}

func (l Rtjc) sendSummary(ctx context.Context, msg string) {
	if !strings.HasPrefix(msg, "⚠") || !l.EnableSummary {
		return
	}

	summaryMsgs, err := l.Summarizer.GetSummariesByMessage(msg)
	if err != nil {
		log.Printf("[WARN] can't get summary, %v", err)
		return
	}

	// By default, rate limit to 15 messages per 2 minutes (1 per 8 sec)
	// Telegram asks 30 sec of waiting after sending 20 messages
	rl := rate.NewLimiter(l.SubmitRateLimit, l.SubmitRateBurst)
	for i, sumMsg := range summaryMsgs {
		if sumMsg == "" {
			log.Printf("[WARN] empty summary item #%d for %q", i, msg)
			continue
		}
		if err := rl.Wait(ctx); err != nil {
			log.Printf("[WARN] can't wait for rate limit, %v", err)
		}
		if err := l.Submitter.SubmitHTML(ctx, sumMsg, false); err != nil {
			log.Printf("[WARN] can't send summary, %v", err)
		}
	}
}

func (l Rtjc) isPinned(msg string) (ok bool, m string) {
	cleanedMsg := strings.TrimSpace(msg)
	cleanedMsg = strings.TrimSuffix(cleanedMsg, "\n")

	for k, v := range pinned {
		if strings.EqualFold(cleanedMsg, k) {
			resMsg := v
			if strings.TrimSpace(resMsg) == "" {
				resMsg = msg
			}
			return true, resMsg
		}
	}
	return false, msg
}
