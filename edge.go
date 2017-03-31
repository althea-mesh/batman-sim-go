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
	satDuration := time.Duration(bits*(1000000/edge.Throughput)) * time.Microsecond

	log.Println("saturated for", satDuration)
	time.Sleep(satDuration)
	log.Println("unsaturated")

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
	}
}
