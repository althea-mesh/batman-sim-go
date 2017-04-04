package main

import (
	"log"
	"sync"
	"time"
)

type Edge struct {
	Throughput  int
	Destination *Node
	sat         bool
	mut         sync.Mutex
}

func (edge *Edge) Saturate(bytes int) {
	edge.mut.Lock()
	edge.sat = true
	edge.mut.Unlock()

	bits := bytes * 8
	satNum := bits * (1000000 / edge.Throughput)
	// satDuration := time.Duration(bits*(1000000/edge.Throughput)) * time.Microsecond

	for i := 0; i < satNum*10000; i++ {
		time.Sleep(time.Microsecond * 1)
	}

	edge.mut.Lock()
	edge.sat = false
	edge.mut.Unlock()
}

func (edge *Edge) IsSaturated() bool {
	edge.mut.Lock()
	sat := edge.sat
	edge.mut.Unlock()

	return sat
}

func (edge *Edge) SendPacket(packet Packet) {
	if !edge.IsSaturated() {
		go edge.Saturate(len(packet.Payload))
		edge.Destination.PacketChannel <- packet
	} else {
		log.Println("DROPPED")
	}
}
