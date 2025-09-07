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
	delay, err := ex.QueryParamInt("delay", 1)
	if err != nil {
		return response.BadRequest("Invalid delay value")
	}
	if delay < 1 {
		return response.BadRequest("Delay must be greater than 0")
	}
	if delay > 10 {
		return response.BadRequest("Delay must be less than 10")
	}

	count, err := ex.QueryParamInt("count", 10)
	if err != nil {
		return response.BadRequest("Invalid count value")
	}
	if count < 1 {
		return response.BadRequest("Count must be greater than 0")
	}
	if count > 100 {
		return response.BadRequest("Count must be less than 100")
	}

	return response.Response{
		Header: map[string][]string{
			"Cache-Control": {"no-store"},
			c.ContentType:   {"text/event-stream"},
		},
		Writer: func(w response.BodyWriter) {
			for id := range count {
				err := w.Write(strings.Join(pingMessage(id+1), "\n") + "\n\n")
				if err != nil {
					log.Printf("Error writing to response: %v\n", err)
				}
				time.Sleep(time.Duration(delay) * time.Second)
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
