package sse

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/sharat87/httpbun/c"
	"github.com/sharat87/httpbun/ex"
	"github.com/sharat87/httpbun/response"
)

var RouteList = []ex.Route{
	ex.NewRoute("/sse", handleServerSentEvents),
}

func handleServerSentEvents(ex *ex.Exchange) response.Response {
	delay, err := ex.QueryInt("delay", 1, 1, 10)
	if err != nil {
		return response.BadRequest("%s", err.Error())
	}

	count, err := ex.QueryInt("count", 10, 1, 100)
	if err != nil {
		return response.BadRequest("%s", err.Error())
	}

	return response.Response{
		Header: map[string][]string{
			"Cache-Control": {"no-store"},
			c.ContentType:   {"text/event-stream"},
		},
		Writer: func(w response.BodyWriter) {
			for id := range count {
				if id > 0 {
					select {
					case <-ex.Request.Context().Done():
						// The client has disconnected.
						return
					case <-time.After(time.Duration(delay) * time.Second):
					}
				}
				err := w.Write(strings.Join(pingMessage(id+1), "\n") + "\n\n")
				if err != nil {
					// The client has most likely disconnected.
					log.Printf("Error writing to response: %v\n", err)
					return
				}
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
