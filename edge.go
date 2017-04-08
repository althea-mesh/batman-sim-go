package main

import "time"

const quantize = 10000

type Edge struct {
	Throughput    int
	PacketChannel *chan (Packet)
	lastPacket    PacketRecord
}

func (edge *Edge) SendPacket(packet Packet) {
	bits := edge.lastPacket.Bytes * 8
	satDuration := time.Duration(bits*1000/edge.Throughput) * time.Millisecond

	if time.Since(edge.lastPacket.Time) > satDuration {
		*edge.PacketChannel <- packet
		edge.lastPacket = PacketRecord{
			Time:  time.Now(),
			Bytes: len(packet.Payload),
		}
	}
}
