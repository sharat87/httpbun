package sse

import (
	"fmt"
	"github.com/sharat87/httpbun/c"
	"github.com/sharat87/httpbun/exchange"
	"github.com/sharat87/httpbun/response"
	"log"
	"strings"
	"time"
)

var Routes = map[string]exchange.HandlerFn{
	`/sse`: handleServerSentEvents,
}

func handleServerSentEvents(ex *exchange.Exchange) response.Response {
	return response.Response{
		Header: map[string][]string{
			"Cache-Control": {"no-store"},
			c.ContentType:   {"text/event-stream"},
		},
		Writer: func(w response.BodyWriter) {
			for id := range 10 {
				err := w.Write(strings.Join(pingMessage(id+1), "\n") + "\n\n")
				if err != nil {
					log.Printf("Error writing to response: %v\n", err)
				}
				time.Sleep(1 * time.Second)
			}
		},
	}
}

func pingMessage(id int) []string {
	return []string{
		"event: ping",
		fmt.Sprintf("id: %v", id),
		"data: a ping event",
	}
}
