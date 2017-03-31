package main

/*
When you start receiving packets from a certain source, start an interval timer.

On that interval, send the source an ack message containing a start time, an end time,
and the number of bytes you have received from that source during the time period.

When you receive such an ack from a destination, discard it if you have switched next hops to
that destination after the ack's start time.
(This is because the ack will not be relevant to the new next hop)

Next, compare the number of bytes reported by the ack to the number of bytes you have sent to
that destination during the ack's time period.

Calculate packet loss percentage using these numbers. If the packet loss percentage computed
from the ack is over some threshold lower than the packet loss percentage reported by the next hop,
adjust the packet loss percentage through that next hop.
The details of this adjustment are covered elsewhere.

Forward the ack to the next hop for that destination.

When you receive a forwarded ack, follow the same procedure.
*/

import (
	"encoding/json"
	"errors"
	"log"
	"time"
)

const (
	timeMultiplier = 1000
	hopPenalty     = 15 / 255
)

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
	Address string
	NextHop struct {
		Address      string
		Throughput   int
		TimeSwitched time.Time
	}
	OgmSequence  int
	PacketsSent  map[string]PacketRecords // indexed by source address
	AcksReceived []Ack
}

type PacketRecords []PacketRecord

type Source struct {
	LastAckSent Ack
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

type Packet struct {
	Type        string
	Source      string
	Destination string
	Payload     []byte
}

type OGM struct {
	Sequence           int
	DestinationAddress string
	SenderAddress      string
	Throughput         int
	Timestamp          int
}

func main() {
	log.SetFlags(log.Lmicroseconds)
	a := Node{
		Address:       "A",
		PacketChannel: make(chan (Packet)),
	}

	b := Node{
		Address:       "B",
		PacketChannel: make(chan (Packet)),
	}

	aToB := Edge{
		Destination: &b,
		Throughput:  1000,
	}

	bToA := Edge{
		Destination: &a,
		Throughput:  1000,
	}

	a.Neighbors = map[string]Neighbor{
		"B": Neighbor{
			Edge: &aToB,
		},
	}

	b.Neighbors = map[string]Neighbor{
		"A": Neighbor{
			Edge: &bToA,
		},
	}

	go a.Listen()
	go b.Listen()

	a.SendSpeedTest("B", time.Microsecond*500, 100)
	time.Sleep(time.Minute)
}

func (node *Node) SendPacket(packet Packet) {
	dest, exists := node.Destinations[packet.Destination]
	if exists {
		dest.PacketsSent[packet.Source] = append(
			dest.PacketsSent[packet.Source],
			PacketRecord{
				Bytes:       len(packet.Payload),
				Time:        time.Now(),
				Source:      packet.Source,
				Destination: packet.Destination,
			},
		)
		address := dest.NextHop.Address
		node.Neighbors[address].Edge.SendPacket(packet)
	}
}

func (node *Node) Listen() {
	for {
		packet := <-node.PacketChannel

		if packet.Source == node.Address {
			node.HandlePacket(packet)
		} else {
			node.SendPacket(packet)
		}
	}
}

func (node *Node) HandlePacket(packet Packet) {
	var err error
	switch packet.Type {
	case "OGM":
		err = node.HandleOGM(packet.Payload)
	case "ACK":
		err = node.HandleAck(packet.Payload)
	default:
		log.Println(string(packet.Payload))
	}

	if err != nil {
		log.Println(node.Address, err)
	}
}

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
	err := json.Unmarshal(payload, ack)
	if err != nil {
		return err
	}
	dest, exists := node.Destinations[ack.Destination]

	if !exists {
		return nil
	}

	if dest.NextHop.TimeSwitched.After(ack.StartTime) {
		return nil
	}

	bytesSent := dest.PacketsSent[ack.Destination].SumBytes(ack.StartTime, ack.EndTime)
	packetLoss := float64(bytesSent / ack.BytesReceived)

	return nil
}

func (prs PacketRecords) SumBytes(start time.Time, end time.Time) int {
	acc := 0
	for _, pr := range prs {
		acc += pr.Bytes
	}

	return acc
}

func (node *Node) HandleOGM(payload []byte) error {
	ogm := OGM{}
	err := json.Unmarshal(payload, ogm)
	if err != nil {
		return err
	}

	if ogm.DestinationAddress == node.Address {
		return nil
	}

	adjustedOGM, err := node.AdjustOGM(ogm)
	if err != nil {
		return err
	}

	err = node.UpdateDestination(*adjustedOGM)
	if err != nil {
		return err
	}

	node.RebroadcastOGM(*adjustedOGM)
	return nil
}

func (node *Node) AdjustOGM(ogm OGM) (*OGM, error) {
	/* Update the received throughput metric to match the link
	 * characteristic:
	 *  - If this OGM traveled one hop so far (emitted by single hop
	 *    neighbor) the path throughput metric equals the link throughput.
	 *  - For OGMs traversing more than one hop the path throughput metric is
	 *    the smaller of the path throughput and the link throughput.
	 */
	neighbor, exists := node.Neighbors[ogm.SenderAddress]
	if !exists {
		return nil, errors.New("OGM not sent from neighbor")
	}

	if ogm.DestinationAddress == ogm.SenderAddress {
		ogm.Throughput = neighbor.Throughput
	} else {
		if neighbor.Throughput < ogm.Throughput {
			ogm.Throughput = neighbor.Throughput
		}
	}

	ogm.Throughput = ogm.Throughput - (ogm.Throughput * hopPenalty)
	return &ogm, nil
}

func (node *Node) UpdateDestination(ogm OGM) error {
	dest, exists := node.Destinations[ogm.DestinationAddress]

	if !exists {
		dest := Destination{
			OgmSequence: ogm.Sequence,
			Address:     ogm.DestinationAddress,
		}
		dest.NextHop.Address = ogm.SenderAddress
		dest.NextHop.Throughput = ogm.Throughput

		node.Destinations[ogm.DestinationAddress] = dest
		return nil
	}

	if ogm.Sequence > dest.OgmSequence {
		return errors.New("ogm sequence too low")
	}

	if dest.NextHop.Throughput < ogm.Throughput {
		dest.NextHop.Throughput = ogm.Throughput
		if dest.NextHop.Address != ogm.SenderAddress {
			dest.NextHop.Address = ogm.SenderAddress
			dest.NextHop.TimeSwitched = time.Now()
		}
	}

	return nil
}

func (node *Node) RebroadcastOGM(ogm OGM) error {
	ogm.SenderAddress = node.Address
	payload, err := json.Marshal(ogm)
	if err != nil {
		return err
	}
	for _, neighbor := range node.Neighbors {
		node.SendPacket(Packet{
			Type:        "OGM",
			Source:      node.Address,
			Destination: neighbor.Address,
			Payload:     payload,
		})
	}
	return nil
}
