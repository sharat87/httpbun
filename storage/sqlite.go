package storage

import (
	"net/http"
	"time"
)

type SqliteStorage struct {
	data map[string]*EntryQueue
}

func NewSqliteStorage() *SqliteStorage {
	st := &SqliteStorage{
		data: map[string]*EntryQueue{},
	}
	st.StartAutoDeletes()
	return st
}

func (st *SqliteStorage) PushRequestToInbox(name string, request http.Request) {
	if _, isExists := st.data[name]; !isExists {
		st.data[name] = NewSizedQueue()
	}
	st.data[name].Push(Entry{
		Protocol: request.Proto,
		Scheme:   request.URL.Scheme,
		Host:     request.Host,
		Path: request.URL.Path,
		Method:   request.Method,
		Params:   request.URL.Query(),
		Headers:  request.Header,
		Fragment: request.URL.Fragment,
		PushedAt: time.Now().UTC(),
	})
}

func (st *SqliteStorage) GetFromInbox(name string) []Entry {
	eq, isExists := st.data[name]
	if isExists {
		return eq.toSlice()
	} else {
		return []Entry{}
	}
}

func (st *SqliteStorage) StartAutoDeletes() {
	st.DoAutoDelete()
}

func (st *SqliteStorage) DoAutoDelete() {
	cutoff := time.Now().UTC().Add(-1 * time.Minute)
	for _, eq := range st.data {
		for i := SIZE; i > 0; i-- {
			entry := eq.Peek()
			if entry == nil {
				break
			}
			if entry.PushedAt.Before(cutoff) {
				eq.Pop()
			}
		}
	}
	time.AfterFunc(time.Minute, st.DoAutoDelete)
}
