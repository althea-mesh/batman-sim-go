package main

import (
	"log"
	"math/rand"
	"sync"
	"time"
)

const quantize = 10000

type Edge struct {
	Throughput    int
	PacketChannel *chan (Packet)
	lastPacket    PacketRecord
	mut           sync.Mutex
}

func (edge *Edge) SendPacket(packet Packet) {
	edge.mut.Lock()
	lastBits := edge.lastPacket.Bytes * 8
	lastTime := edge.lastPacket.Time
	throughput := edge.Throughput
	edge.mut.Unlock()

	satDuration := (time.Duration(lastBits*1000/throughput) * time.Millisecond) + 1*time.Millisecond

	rand.Seed(time.Now().UnixNano())
	random := rand.Float32()
	frac := float32(time.Since(lastTime)) / float32(satDuration)

	if frac > random {
		edge.mut.Lock()
		*edge.PacketChannel <- packet
		edge.lastPacket = PacketRecord{
			Time:  time.Now(),
			Bytes: len(packet.Payload),
		}
		edge.mut.Unlock()
		log.Printf("packet sent with probability %v > %v", frac, random)
	} else {
		log.Printf("packet dropped with probability %v > %v", frac, random)
	}
}
