package main

import (
	"sync"
	"time"
)

type Link struct {
	sat bool
	mut *sync.Mutex
}

func MakeLink(throughput int) Link {
	return Link{false, throughput, &sync.Mutex{}}
}

func (l *Link) Saturate(i int) {
	l.mut.Lock()
	l.sat = true
	l.mut.Unlock()

	time.Sleep(time.Duration(i) * time.Second)

	l.mut.Lock()
	l.sat = false
	l.mut.Unlock()
}

func (l *Link) IsSaturated() bool {
	l.mut.Lock()
	sat := l.sat
	l.mut.Unlock()

	return sat
}
