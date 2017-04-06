package main

import (
	"encoding/json"
	"errors"
	"log"
	"time"
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
	Address      string
	NextHop      NextHop
	OgmSequence  int
	PacketsSent  map[string]PacketRecords // indexed by source address
	AcksReceived []Ack
}

type NextHop struct {
	Address       string
	PacketSuccess float64
	TimeSwitched  time.Time
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

func (node *Node) SendPacket(packet Packet) {
	dest, exists := node.Destinations[packet.Destination]
	if exists {
		packetsSent := dest.PacketsSent[packet.Source]
		packetsSent = append(
			dest.PacketsSent[packet.Source],
			PacketRecord{
				Bytes:       len(packet.Payload),
				Time:        time.Now(),
				Source:      packet.Source,
				Destination: packet.Destination,
			},
		)

		if len(packetsSent) > 100 {
			packetsSent = packetsSent[:100]
		}

		dest.PacketsSent[packet.Source] = packetsSent

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
			Payload:     []byte("aoaoaooaoaoaoaoaoa"),
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
	packetSuccess := float64(ack.BytesReceived / bytesSent)

	log.Println(packetSuccess)

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
	neighbor, exists := node.Neighbors[ogm.SenderAddress]
	if !exists {
		return nil, errors.New("OGM not sent from neighbor")
	}

	ogm.PacketSuccess = ogm.PacketSuccess * neighbor.PacketSuccess * hopMultiplier
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
		dest.NextHop.PacketSuccess = ogm.PacketSuccess

		node.Destinations[ogm.DestinationAddress] = dest
		return nil
	}

	if ogm.Sequence <= dest.OgmSequence {
		return errors.New("ogm sequence too low")
	}

	if dest.NextHop.PacketSuccess < ogm.PacketSuccess {
		dest.NextHop.PacketSuccess = ogm.PacketSuccess

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
