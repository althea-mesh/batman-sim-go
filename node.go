package main

import (
	"encoding/json"
	"log"
	"time"
)

/*
When you start receiving packets from a certain source, start an interval timer.

On that interval, send the source an ack message containing a start time, an end time,
and the number of bytes you have received from that source during the time period.

When you receive such an ack from a destination, discard it if you have switched next hops to
that destination after the ack's start time.
(This is because the ack will not be relevant to the new next hop)

Next, compare the number of bytes reported by the ack to the number of bytes you have sent to
that destination during the ack's time period. Calculate packet success percentage
using these numbers.

If the packet success percentage computed from the ack is over some threshold lower than the
packet success percentage reported by the next hop, adjust the packet success percentage
through that next hop.
The details of this adjustment are covered elsewhere.

Forward the ack to the next hop for that destination.

When you receive a forwarded ack, follow the same procedure.
*/

type Node struct {
	Address       string
	Sources       map[string]Source
	Destinations  map[string]Destination
	Neighbors     map[string]Neighbor
	OgmSequence   int
	PacketChannel chan (Packet)
}

type Neighbor struct {
	Address       string
	PacketSuccess float64
	Edge          *Edge
}

type Destination struct {
	Address      string
	NextHop      NextHop
	OgmSequence  int
	PacketsSent  PacketRecords // indexed by source address
	AcksReceived []Ack
}

type NextHop struct {
	Address       string
	PacketSuccess float64
	TimeSwitched  time.Time
}

type PacketRecords []PacketRecord

type Source struct {
	Address       string
	LastAckTime   time.Time
	BytesReceived int
}

type Ack struct {
	BytesReceived int
	StartTime     time.Time
	EndTime       time.Time
	Source        string
	Destination   string // destination of the data packets, not the ack
}

type PacketRecord struct {
	Bytes       int
	Time        time.Time
	Source      string
	Destination string
}

const ackInterval = 5 * time.Second

// SendPacket looks for a destination in the node's routing table.
// If the destination is found, information about the packet is
// recorded in the node's PacketsSent list, and the packet is sent
// to the destination's next hop.
func (node *Node) SendPacket(packet Packet) {
	dest, exists := node.Destinations[packet.Destination]
	if exists {
		// packetsSent := dest.PacketsSent
		dest.PacketsSent = append(
			dest.PacketsSent,
			PacketRecord{
				Bytes:       len(packet.Payload),
				Time:        time.Now(),
				Source:      packet.Source,
				Destination: packet.Destination,
			},
		)

		// Let's wait until the rest of it works
		// if len(packetsSent) > 100 {
		// 	packetsSent = packetsSent[:100]
		// }

		address := dest.NextHop.Address
		node.Neighbors[address].Edge.SendPacket(packet)

		node.Destinations[packet.Destination] = dest
	}
}

// Listen receives packets and either forwards them or processes them
// locally with HandlePacket if they are addressed to this node.
func (node *Node) Listen() {
	for {
		packet := <-node.PacketChannel

		if packet.Destination == node.Address {
			node.HandlePacket(packet)
		} else {
			node.SendPacket(packet)
		}
	}
}

// HandlePacket updates the BytesReceived property for the given source
// and calls different handlers based on the type of packet.
func (node *Node) HandlePacket(packet Packet) {
	node.UpdateSourceRecord(packet)
	var err error
	switch packet.Type {
	case "OGM":
		err = node.HandleOGM(packet.Payload)
	case "ACK":
		err = node.HandleAck(packet.Payload)
	default:
		log.Printf("%v got packet of length: %v", node.Address, len(packet.Payload))
	}

	if err != nil {
		log.Printf("error on %v: %v not found.", node.Address, err)
	}
}

func (node *Node) UpdateSourceRecord(packet Packet) {
	source, exists := node.Sources[packet.Source]
	if exists {
		source.BytesReceived = source.BytesReceived + len(packet.Payload)
		if time.Since(source.LastAckTime) > ackInterval {
			node.SendAck(source)
			source.BytesReceived = 0
			source.LastAckTime = time.Now()
		}
	} else {
		source = Source{
			Address:       packet.Source,
			BytesReceived: len(packet.Payload),
			LastAckTime:   time.Now(),
		}
	}

	node.Sources[packet.Source] = source
}

// SendAck Sends an Ack to a source, and resets the BytesReceived counter
func (node *Node) SendAck(source Source) error {
	payload, err := json.Marshal(Ack{
		BytesReceived: source.BytesReceived,
		StartTime:     source.LastAckTime,
		EndTime:       time.Now(),
		Source:        source.Address,
		Destination:   node.Address,
	})

	if err != nil {
		return err
	}

	node.SendPacket(Packet{
		Destination: source.Address,
		Source:      node.Address,
		Type:        "ACK",
		Payload:     payload,
	})

	return nil
}

// SendSpeedTest sends a packet of a certain size at a certain interval to attempt to
// saturate a connection
func (node *Node) SendSpeedTest(destination string, interval time.Duration, numBytes int) {
	for range time.Tick(interval) {
		node.SendPacket(Packet{
			Type:        "DATA",
			Destination: destination,
			Source:      node.Address,
			Payload:     make([]byte, numBytes),
		})
	}
}

func (node *Node) HandleAck(payload []byte) error {
	ack := Ack{}
	err := json.Unmarshal(payload, &ack)
	if err != nil {
		return err
	}
	dest, exists := node.Destinations[ack.Destination]

	if !exists {
		return nil
	}

	// When you receive an ack from a destination, discard it if you have switched the next hop of
	// that destination after the ack's start time.
	// (This is because the ack will not be relevant to the new next hop)
	if dest.NextHop.TimeSwitched.After(ack.StartTime) {
		return nil
	}

	// Next, compare the number of bytes reported by the ack to the number of bytes you have sent to
	// that destination during the ack's time period.
	bytesSent := dest.PacketsSent.SumBytes(ack.StartTime, ack.EndTime)

	// Calculate packet success percentage using these numbers.

	packetSuccess := float32(ack.BytesReceived) / float32(bytesSent)

	// If the packet success percentage computed from the ack is over some threshold lower than the
	// packet success percentage reported by the next hop, adjust the packet success percentage
	// through that next hop.
	log.Printf(
		"ack.BytesReceived: %v, bytesSent: %v, packet success: %v",
		ack.BytesReceived, bytesSent, packetSuccess,
	)

	return nil
}

func (prs PacketRecords) SumBytes(start time.Time, end time.Time) int {
	acc := 0
	for _, pr := range prs {
		if pr.Time.After(start) && pr.Time.Before(end) {
			acc += pr.Bytes
		}
	}

	return acc
}
