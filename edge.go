package main

import (
	"sync"
	"time"
)

type Edge struct {
	Throughput    int
	PacketChannel *chan (Packet)
	sat           bool
	mut           sync.Mutex
}

func (edge *Edge) saturate(bytes int) {
	edge.mut.Lock()
	edge.sat = true
	edge.mut.Unlock()

	bits := bytes*8 + 20
	satDuration := time.Duration(bits*(1000000/edge.Throughput)) * time.Microsecond

	time.Sleep(satDuration)

	edge.mut.Lock()
	edge.sat = false
	edge.mut.Unlock()
}

func (edge *Edge) isSaturated() bool {
	edge.mut.Lock()
	sat := edge.sat
	edge.mut.Unlock()

	return sat
}

func (edge *Edge) SendPacket(packet Packet) {
	if !edge.isSaturated() {
		go edge.saturate(len(packet.Payload))
		*edge.PacketChannel <- packet
	}
}
