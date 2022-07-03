package storage

import (
	"net/http"
	"time"
)

type Storage interface {
	PushRequestToInbox(name string, request http.Request)
	GetFromInbox(name string) []Entry
}

// TODO: Handle binary body.

type Entry struct {
	Inbox    string              `json:"inbox"`
	Protocol string              `json:"protocol"`
	Scheme   string              `json:"scheme"`
	Host     string              `json:"host"`
	Path     string              `json:"path"`
	Method   string              `json:"method"`
	Params   map[string][]string `json:"params"`
	Headers  map[string][]string `json:"headers"`
	Fragment string              `json:"fragment"`
	Body     string              `json:"body"`
	PushedAt time.Time           `json:"pushedAt" bson:"pushedAt"`
}
