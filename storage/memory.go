package storage

import (
	"net/http"
	"time"
)

type MemoryStorage struct {
	data map[string]*EntryQueue
}

func NewMemoryStorage() *MemoryStorage {
	st := &MemoryStorage{
		data: map[string]*EntryQueue{},
	}
	st.StartAutoDeletes()
	return st
}

func (st *MemoryStorage) PushRequestToInbox(name string, request http.Request) {
	if _, isExists := st.data[name]; !isExists {
		st.data[name] = NewSizedQueue()
	}
	st.data[name].Push(Entry{
		Protocol: request.Proto,
		Scheme:   request.URL.Scheme,
		Host:     request.Host,
		Path:     request.URL.Path,
		Method:   request.Method,
		Params:   request.URL.Query(),
		Headers:  request.Header,
		Fragment: request.URL.Fragment,
		PushedAt: time.Now().UTC(),
	})
}

func (st *MemoryStorage) GetFromInbox(name string) []Entry {
	eq, isExists := st.data[name]
	if isExists {
		return eq.toSlice()
	} else {
		return []Entry{}
	}
}

func (st *MemoryStorage) StartAutoDeletes() {
	st.DoAutoDelete()
}

func (st *MemoryStorage) DoAutoDelete() {
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

const SIZE = 10

type EntryQueue struct {
	slots [SIZE]*Entry
	first uint8
	last  uint8
}

func NewSizedQueue() *EntryQueue {
	return &EntryQueue{}
}

func (q *EntryQueue) Push(entry Entry) {
	if q.first == q.last {
		if q.slots[q.first] == nil {
			q.slots[q.last] = &entry
		} else {
			q.last = (q.last + 1) % SIZE
			q.slots[q.last] = &entry
		}
	} else {
		q.last = (q.last + 1) % SIZE
		q.slots[q.last] = &entry
		if q.first == q.last {
			q.first = (q.first + 1) % SIZE
		}
	}
}

func (q *EntryQueue) Peek() *Entry {
	return q.slots[q.first]
}

func (q *EntryQueue) Pop() *Entry {
	if q.first == q.last && q.slots[q.last] == nil {
		return nil
	}
	entry := q.slots[q.first]
	q.slots[q.first] = nil
	if q.first != q.last {
		q.first = (q.first + 1) % SIZE
	}
	return entry
}

func (q *EntryQueue) toSlice() []Entry {
	out := []Entry{}
	count := 0
	i := q.last
	for q.slots[i] != nil && count < SIZE {
		out = append(out, *q.slots[i])
		if i == 0 {
			i = SIZE - 1
		} else {
			i = i - 1
		}
		count += 1
	}
	return out
}
