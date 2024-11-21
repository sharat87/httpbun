package sse

import (
	"fmt"
	"github.com/sharat87/httpbun/c"
	"github.com/sharat87/httpbun/exchange"
	"log"
	"net/http"
	"strings"
	"time"
)

var Routes = map[string]exchange.HandlerFn{
	`/sse`: handleServerSentEvents,
}

func handleServerSentEvents(ex *exchange.Exchange) {
	ex.ResponseWriter.Header().Set("Cache-Control", "no-store")
	ex.ResponseWriter.Header().Set(c.ContentType, "text/event-stream")
	ex.ResponseWriter.WriteHeader(http.StatusOK)

	if f, ok := ex.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	} else {
		log.Println("Flush not available. Dripping and streaming not supported on this platform.")
	}

	for id := range 10 {
		time.Sleep(1 * time.Second)
		_, err := fmt.Fprint(ex.ResponseWriter, strings.Join(pingMessage(id+1), "\n")+"\n\n")
		if err != nil {
			return
		}
		if f, ok := ex.ResponseWriter.(http.Flusher); ok {
			f.Flush()
		} else {
			log.Println("Flush not available. Dripping and streaming not supported on this platform.")
		}
	}
}

func pingMessage(id int) []string {
	return []string{
		"event: ping",
		fmt.Sprintf("id: %v", id),
		"data: a ping event",
	}
}
